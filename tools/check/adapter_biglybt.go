//go:build !windows

// adapter_biglybt - the CLIENT-09 capture adapter on Linux, `bit-check adapter
// biglybt`.
//
// The contract is scripts/capture/adapters/README.md. ⚠ The Windows half
// registers the same name from adapter_biglybt_windows.go; one platform builds
// one of them. What either half decides without a process in it is biglybt.go,
// which `bit-check biglybt --selftest` asks on any host.
//
// ⭐ DRIVEN HEADLESS BY THE VENDOR'S OWN CONSOLE UI. `com.biglybt.ui.Main
// --ui=console <torrent>` starts the core with no display, adds the torrent
// from the command line, and keeps running when the console's stdin reaches
// end of file. Nothing remote is enabled: no telnet UI, no Web UI, no plugin
// listening for commands, so the adapter holds no credential. `stop` sends
// SIGTERM, and the core's own shutdown hook stops every download, which
// announces `stopped`; measured at three seconds on 2026-10-21.
//
// -- the routes, measured on 2026-10-21 -----------------------------------------
//
// ⛔ `release`: the vendor's `GitHub_BiglyBT_unix.tar.gz` from `v4.1.0.0`,
// fetched by `install-rootless --fetch` under the vendor's
// `GitHub_sha256sum.txt` and the listing's digest, and extracted whole into
// `<prefix>/opt/biglybt`. It carries no JRE.
// ⛔ `source`: the `v4.1.0.0` tag from the repository's own refs, compiled by
// the host's JDK as upstream's Maven profile for JDK 9 and later compiles it,
// and jarred into the same place beside the tag's own `commons-cli.jar`.
// ⚠ Upstream's Maven build also needs a repository off GitHub for SWT 4.9; the
// tag carries its SWT jars, so `javac` alone is the build, and the recipe is
// recorded in `build-deps.txt`.
// ⛔ BOTH RUN ON THE HOST'S JVM: `BIT_IDS_BIGLYBT_JAVA`, else `$JAVA_HOME`, else
// `java` on PATH. On a runner that is one JDK on both lanes, so the routes
// differ in how BiglyBT arrived and not in what runs it.
// ⛔ NO PACKAGE ROUTE ON LINUX. Ubuntu carries no BiglyBT, Flathub and Snap need
// a daemon and privilege, and `ACQ-06` keeps a privileged route off the
// capture path. The vendor's install4j `.sh` installer is the tarball's
// listing and delivery again, so it is not a second route.
//
// Exit codes: 0 done, 1 refused, 2 could not run.
package main

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"io/fs"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"
)

func init() { registerAdapter(bbTarget, shAdapter(bbTarget, biglybtAdapter)) }

// bbInstall is where both routes place the build. ⛔ THE PREFIX IS ASKED FOR.
func bbInstall(k *adapterKit, quiet bool) (string, bool) {
	p, ok := k.prefix(quiet)
	if !ok {
		return "", false
	}
	return filepath.Join(p, "opt", "biglybt"), true
}

// bbInstalledJar is the installed jar, or "": an override, then the prefix, and
// ⛔ NEVER PATH.
func bbInstalledJar(k *adapterKit) string {
	if v := os.Getenv("BIT_IDS_BIGLYBT_HOME"); v != "" {
		return filepath.Join(v, bbJar)
	}
	dir, ok := bbInstall(k, true)
	if !ok {
		return ""
	}
	if jar := filepath.Join(dir, bbJar); isFile(jar) {
		return jar
	}
	return ""
}

// bbJava is the host's JVM: an override, then JAVA_HOME, then PATH.
func bbJava() (string, string) {
	if v := os.Getenv("BIT_IDS_BIGLYBT_JAVA"); v != "" {
		return v, "BIT_IDS_BIGLYBT_JAVA"
	}
	if h := os.Getenv("JAVA_HOME"); h != "" {
		if j := filepath.Join(h, "bin", "java"); isExec(j) {
			return j, "JAVA_HOME"
		}
	}
	if j, ok := commandV("java"); ok {
		return j, "PATH"
	}
	return "", ""
}

// bbJDKTool is `javac` or `jar` from the JDK the JVM belongs to.
func bbJDKTool(java, tool string) string {
	real, err := filepath.EvalSymlinks(java)
	if err != nil {
		real = java
	}
	if t := filepath.Join(filepath.Dir(real), tool); isExec(t) {
		return t
	}
	return ""
}

func biglybtAdapter(k *adapterKit, sub string, args []string) {
	switch sub {
	case "describe":
		fmt.Printf("target=%s\n", bbTarget)
		fmt.Print("kind=stock\n")
		if jar := bbInstalledJar(k); jar != "" {
			fmt.Printf("binary=%s\n", jar)
		}
		fmt.Print(bbDescribeScheme(bbLinuxAsset))
		fmt.Print("compiles=source\n")
		fmt.Print("routes=release,source\n")

	case "install":
		route, workdir, format := k.installArgs(args, func(r string) (string, bool) { return bbPackageFormat(false, r) })
		switch route {
		case "release":
			bbRelease(k, workdir)
		case "source":
			bbSource(k, workdir)
		default:
			k.cannot("unknown route: " + route)
		}
		k.writePackage(workdir, format)

	case "version":
		jar := bbInstalledJar(k)
		if jar == "" {
			k.cannot(bbJar + " is not installed")
		}
		output, a, err := bbAsk(k, jar)
		if err != nil {
			k.cannot(err.Error())
		}
		k.versionReport(output, a.version)

	case "start":
		bbStart(k, args)

	case "stop":
		bbStop(k, args)
	}
}

// bbAsk runs the probe class on the host's JVM against the installed jar.
// ⛔ THE BUILD SPEAKING, AND A PURE READ: the class is written to a temporary
// directory removed after the call, and `Constants` reads system properties
// and nothing else.
func bbAsk(k *adapterKit, jar string) (string, bbAnswer, error) {
	java, _ := bbJava()
	if java == "" {
		return "", bbAnswer{}, fmt.Errorf("no JVM: BIT_IDS_BIGLYBT_JAVA, JAVA_HOME and PATH name none")
	}
	tmp, err := os.MkdirTemp("", "biglybt-version-")
	if err != nil {
		return "", bbAnswer{}, fmt.Errorf("cannot make a temporary directory")
	}
	defer os.RemoveAll(tmp)
	if err := os.WriteFile(filepath.Join(tmp, bbProbeClass+".class"), bbVersionClass(), 0o644); err != nil {
		return "", bbAnswer{}, fmt.Errorf("cannot write the probe")
	}
	c := exec.Command(java, "-cp", tmp+":"+jar, bbProbeClass)
	c.Env, _ = bbJavaEnv(os.Environ(), "HOME="+tmp)
	output, rc := capture(c, stdio{in: os.DevNull, err: "&1"})
	if rc != 0 {
		return output, bbAnswer{}, fmt.Errorf("the probe exited %d: %s", rc, firstLine(output))
	}
	a, err := bbParseAnswer(output)
	return output, a, err
}

// bbCheckBuilt asks the build the route placed and refuses a version that is
// not the one the route was handed.
func bbCheckBuilt(k *adapterKit, workdir, want, from string) {
	jar := bbInstalledJar(k)
	dir, _ := bbInstall(k, true)
	if jar != filepath.Join(dir, bbJar) {
		k.refuse("the route installed and left no " + bbJar + " in " + dir)
	}
	output, a, err := bbAsk(k, jar)
	_ = writeText(filepath.Join(workdir, "install-version.log"), output+"\n")
	if err != nil {
		k.refuse(err.Error() + "; install-version.log has the whole answer")
	}
	if want != "" && a.version != want {
		k.refuse(fmt.Sprintf("%s named %s and the build it installed reports %s", from, want, a.version))
	}
}

// bbReplace places a staged tree at the install, replacing what was there.
func bbReplace(k *adapterKit, staged string) {
	dir, ok := bbInstall(k, false)
	if !ok {
		k.cannot("the rootless prefix could not be derived")
	}
	if os.RemoveAll(dir) != nil || os.Rename(staged, dir) != nil {
		k.refuse("the build could not be placed at " + dir)
	}
}

// bbStage is a fresh staging directory beside the install.
func bbStage(k *adapterKit) string {
	dir, ok := bbInstall(k, false)
	if !ok {
		k.cannot("the rootless prefix could not be derived")
	}
	staged := dir + ".new"
	if os.RemoveAll(staged) != nil || !mkdirP(staged) {
		k.refuse("this user cannot create " + staged + "; a rootless install writes nowhere else")
	}
	return staged
}

// bbRelease is the release route. ⛔ The URL is resolved before the route is
// cut and handed in; the fetch and the digest are install-rootless's one call.
func bbRelease(k *adapterKit, workdir string) {
	url := os.Getenv("BIT_IDS_RELEASE_URL")
	if url == "" {
		k.cannot("the release route needs BIT_IDS_RELEASE_URL, resolved before the route was cut")
	}
	want := ""
	if res := os.Getenv("BIT_IDS_RELEASE_RESOLUTION"); res != "" {
		body, _ := os.ReadFile(res)
		want = recordField(body, "selected_version")
	}
	k.need(func(string) string { return "curl is not on this host" }, "curl")
	archive := filepath.Join(workdir, "biglybt_unix.tar.gz")
	fetch := append([]string{"--fetch", "--url", url, "--into", archive}, k.releaseDisposition()...)
	fetch = append(fetch, "--report", filepath.Join(workdir, "rootless-install.txt"))
	k.rootlessRelease(workdir, "the release route could not fetch the vendor's tarball", fetch...)
	staged := bbStage(k)
	n, err := bbUntar(archive, staged)
	if err != nil {
		_ = os.RemoveAll(staged)
		k.refuse("the vendor's tarball could not be extracted: " + err.Error())
	}
	if !isFile(filepath.Join(staged, bbJar)) || !isFile(filepath.Join(staged, bbCLIJar)) {
		_ = os.RemoveAll(staged)
		k.refuse("the vendor's tarball carries no " + bbJar + " with " + bbCLIJar + " beside it")
	}
	bbReplace(k, staged)
	dir, _ := bbInstall(k, true)
	sum, _ := fileSHA256(archive)
	jarSum, _ := fileSHA256(filepath.Join(dir, bbJar))
	note := fmt.Sprintf("archive=%s\narchive_sha256=%s\nmembers=%d\ninstall=%s\njar_sha256=%s\n", filepath.Base(archive), sum, n, dir, jarSum)
	if writeText(filepath.Join(workdir, "extract.txt"), note) != nil {
		k.refuse("the route could not record what it extracted")
	}
	_ = os.Remove(archive)
	if err := bbRecordTree(dir, filepath.Join(workdir, "installed-files.txt")); err != nil {
		k.refuse("the installed tree could not be digested: " + err.Error())
	}
	bbCheckBuilt(k, workdir, want, "the release resolution")
}

// bbUntar extracts the vendor's tarball into dir, placing directories and
// regular files only, under biglybt/, and answers how many members it placed.
func bbUntar(archive, dir string) (int, error) {
	f, err := os.Open(archive)
	if err != nil {
		return 0, err
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		return 0, err
	}
	tr := tar.NewReader(gz)
	n := 0
	for {
		h, err := tr.Next()
		if err == io.EOF {
			return n, nil
		}
		if err != nil {
			return n, err
		}
		if h.Typeflag == tar.TypeXGlobalHeader {
			continue
		}
		isDir := h.Typeflag == tar.TypeDir
		rel, err := bbTarMember(h.Name, isDir, h.Typeflag == tar.TypeReg)
		if err != nil {
			return n, err
		}
		target := filepath.Join(dir, filepath.FromSlash(rel))
		if isDir {
			if err := os.MkdirAll(target, 0o755); err != nil {
				return n, err
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return n, err
		}
		mode := os.FileMode(0o644)
		if h.FileInfo().Mode()&0o111 != 0 {
			mode = 0o755
		}
		out, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_EXCL, mode)
		if err != nil {
			return n, err
		}
		_, cerr := io.Copy(out, tr)
		if err := out.Close(); cerr == nil {
			cerr = err
		}
		if cerr != nil {
			return n, cerr
		}
		n++
	}
}

// bbRecordTree writes every file under root with its sha256, sorted by path,
// which is the digest of what a route installed.
func bbRecordTree(root, out string) error {
	var lines []string
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil || !d.Type().IsRegular() {
			return err
		}
		sum, err := fileSHA256(p)
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(root, p)
		lines = append(lines, sum+"  "+filepath.ToSlash(rel))
		return nil
	})
	if err != nil {
		return err
	}
	sort.Slice(lines, func(i, j int) bool { return lines[i][66:] < lines[j][66:] })
	return writeText(out, strings.Join(lines, "\n")+"\n")
}

// bbSource is the source route: the repository's own tag, cloned and compiled.
func bbSource(k *adapterKit, workdir string) {
	tag := os.Getenv("BIT_IDS_SOURCE_TAG")
	if tag == "" {
		k.cannot("the source route needs BIT_IDS_SOURCE_TAG, resolved from this repository's own refs before the route was cut")
	}
	want := bbTagVersion(tag)
	if want == "" {
		k.cannot("the source tag [" + tag + "] names no BiglyBT version")
	}
	k.need(func(string) string { return "the source route clones and git is not on this host" }, "git")
	java, from := bbJava()
	if java == "" {
		k.cannot("the source route compiles with the host's JDK and BIT_IDS_BIGLYBT_JAVA, JAVA_HOME and PATH name no java")
	}
	javac, jarTool := bbJDKTool(java, "javac"), bbJDKTool(java, "jar")
	if javac == "" || jarTool == "" {
		k.cannot("the JVM at " + java + " is not a JDK: no javac or jar beside it")
	}
	log := filepath.Join(workdir, "install.log")
	src := filepath.Join(workdir, "src")
	c := exec.Command("timeout", "600", "git", "clone", "--depth", "1", "--branch", tag, "https://github.com/"+bbRepo+".git", src)
	if rc := run(c, stdio{in: os.DevNull, out: log, err: "&1"}); rc != 0 {
		if rc == 124 {
			k.refuse("the source route's clone of " + tag + " did not finish within 600s")
		}
		k.refuse("the source route could not clone " + tag)
	}
	k.cloneCommit(workdir, tag)

	// ⛔ WHAT IS COMPILED AND WHAT IS COPIED IS ONE DECISION, bbSourceFile.
	var sources, resources []string
	for _, root := range []string{"core/src", "uis/src"} {
		base := filepath.Join(src, root)
		err := filepath.WalkDir(base, func(p string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return err
			}
			rel, _ := filepath.Rel(src, p)
			switch bbSourceFile(rel) {
			case "java":
				sources = append(sources, p)
			case "resource":
				r, _ := filepath.Rel(base, p)
				resources = append(resources, root+"\x00"+r)
			}
			return nil
		})
		if err != nil {
			k.refuse("the tag's sources could not be listed: " + err.Error())
		}
	}
	sort.Strings(sources)
	sort.Strings(resources)
	list := filepath.Join(workdir, "sources.list")
	if writeText(list, strings.Join(sources, "\n")+"\n") != nil {
		k.refuse("the source route could not write its source list")
	}
	classes := filepath.Join(workdir, "classes")
	if os.RemoveAll(classes) != nil || !mkdirP(classes) {
		k.refuse("the source route could not make its class directory")
	}
	env, dropped := bbJavaEnv(os.Environ())
	c = exec.Command("timeout", append([]string{"900", javac}, bbJavacArgs(src, classes, list)...)...)
	c.Env = env
	started := time.Now()
	if rc := run(c, stdio{in: os.DevNull, out: log, outAppend: true, err: "&1"}); rc != 0 {
		run(exec.Command("tail", "-20", log), stdio{out: "&2"})
		if rc == 124 {
			k.refuse("javac did not finish within 900s")
		}
		k.refuse(fmt.Sprintf("javac could not compile %s (exit %d)", tag, rc))
	}
	compiled := time.Since(started).Round(time.Second)
	for _, r := range resources {
		root, rel, _ := strings.Cut(r, "\x00")
		to := filepath.Join(classes, rel)
		if !mkdirP(filepath.Dir(to)) || !cpFile(filepath.Join(src, root, rel), to) {
			k.refuse("the resource " + rel + " could not be copied")
		}
	}
	for _, cp := range bbResourceCopies {
		from := filepath.Join(classes, filepath.FromSlash(cp[0]))
		if !isFile(from) || !cpFile(from, filepath.Join(classes, filepath.FromSlash(cp[1]))) {
			k.refuse("upstream's copy of " + cp[0] + " could not be made")
		}
	}
	manifest := filepath.Join(workdir, "MANIFEST.MF")
	if writeText(manifest, bbManifest) != nil {
		k.refuse("the source route could not write the jar's manifest")
	}
	staged := bbStage(k)
	c = exec.Command(jarTool, "--create", "--file", filepath.Join(staged, bbJar), "--manifest", manifest, "-C", classes, ".")
	c.Env = env
	if run(c, stdio{in: os.DevNull, out: log, outAppend: true, err: "&1"}) != 0 {
		_ = os.RemoveAll(staged)
		k.refuse("the jar could not be made")
	}
	if !cpFile(filepath.Join(src, "core", "lib", bbCLIJar), filepath.Join(staged, bbCLIJar)) {
		_ = os.RemoveAll(staged)
		k.refuse("the tag's " + bbCLIJar + " could not be placed")
	}
	bbReplace(k, staged)
	javacVersion, _ := capture(exec.Command(javac, "-version"), stdio{in: os.DevNull, err: "&1"})
	javaSum, _ := fileSHA256(java)
	cliSum, _ := fileSHA256(filepath.Join(src, "core", "lib", bbCLIJar))
	deps := fmt.Sprintf("java=%s\njava_from=%s\njava_sha256=%s\njavac=%s\njavac_version=%s\njar=%s\n"+
		"recipe=upstream's Maven profile java-9-plus, run as javac: %s excluded, -source 8 -target 8, ISO-8859-1, resources copied, two locale copies made\n"+
		"compile=javac %s\nsources=%d\nresources=%d\ncompile_seconds=%d\ncommons_cli_sha256=%s\njvm_env_dropped=%s\n",
		java, from, javaSum, javac, javacVersion, jarTool, bbSourceExcluded, strings.Join(bbJavacArgs("<src>", "<classes>", "<sources>"), " "),
		len(sources), len(resources), int(compiled.Seconds()), cliSum, strings.Join(dropped, ","))
	if writeText(filepath.Join(workdir, "build-deps.txt"), deps) != nil {
		k.refuse("the source route could not record what it built with")
	}
	_ = os.RemoveAll(classes)
	dir, _ := bbInstall(k, true)
	if err := bbRecordTree(dir, filepath.Join(workdir, "installed-files.txt")); err != nil {
		k.refuse("the installed tree could not be digested: " + err.Error())
	}
	bbCheckBuilt(k, workdir, want, "the tag "+tag)
}

// bbFreePort asks the kernel for a loopback port nothing holds.
func bbFreePort() (int, error) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port, nil
}

func bbStart(k *adapterKit, args []string) {
	if len(args) < 3 {
		k.cannot("start takes <torrent> <workdir> <peer-port>")
	}
	torrent, workdir, port := args[0], args[1], args[2]
	jar := bbInstalledJar(k)
	if jar == "" {
		k.cannot(bbJar + " is not installed")
	}
	if !isNonEmpty(torrent) {
		k.cannot("there is no torrent at " + torrent)
	}
	java, from := bbJava()
	if java == "" {
		k.cannot("no JVM: BIT_IDS_BIGLYBT_JAVA, JAVA_HOME and PATH name none")
	}
	// ⛔ THE JVM IS ASKED FIRST, AND WHAT IT SAID IS KEPT PER SESSION: the
	// record's JVM facts are this session's, and the launch depends on its
	// feature release.
	output, a, err := bbAsk(k, jar)
	if err != nil {
		k.cannot(err.Error())
	}
	_ = appendText(filepath.Join(workdir, "jvm.txt"), fmt.Sprintf("## %s java=%s from=%s\n%s\n", utcNow(), java, from, output))
	config := filepath.Join(workdir, "config")
	downloads := filepath.Join(workdir, "downloads")
	home := filepath.Join(workdir, "home")
	// ⛔ A FRESH CONFIG DIRECTORY PER SESSION, written whole, and asked before it
	// is used: every surface off, or nothing starts.
	if os.RemoveAll(config) != nil || !mkdirP(config, downloads, home) {
		k.cannot("cannot create the config under " + workdir)
	}
	doc, err := bbConfig(downloads, port)
	if err != nil {
		k.cannot(err.Error())
	}
	if msg := bbConfigRefusal(doc); msg != "" {
		k.refuse("the settings would start the build with " + msg)
	}
	if os.WriteFile(filepath.Join(config, "biglybt.config"), doc, 0o600) != nil ||
		writeText(filepath.Join(workdir, "settings.txt"), bbSettingsText(doc)) != nil {
		k.cannot("the settings could not be written")
	}
	instance, err := bbFreePort()
	if err != nil {
		k.cannot("no loopback port is free for the client's instance port")
	}
	if writeText(filepath.Join(workdir, "instance-port"), strconv.Itoa(instance)+"\n") != nil {
		k.cannot("cannot record the instance port")
	}
	dir := filepath.Dir(jar)
	major := bbJavaMajor(a.java)
	launch := bbLaunchArgs(dir, ":", config, instance, major, nil, "--ui=console")
	env, dropped := bbJavaEnv(os.Environ(), "HOME="+home)
	_ = appendText(filepath.Join(workdir, "launch.txt"), fmt.Sprintf("## %s\n%s %s\njvm_env_dropped=%s\n", utcNow(), java,
		strings.Join(launch, " "), strings.Join(dropped, ",")))
	out := filepath.Join(workdir, "client.out")
	var offset int64
	if fi, err := os.Stat(out); err == nil {
		offset = fi.Size()
	}
	session := func() string {
		b, _ := os.ReadFile(out)
		if int64(len(b)) < offset {
			return ""
		}
		return string(b[offset:])
	}
	c := exec.Command(java, launch...)
	c.Env = env
	n, err := startBackground(c, stdio{out: out, outAppend: true, err: "&1", dir: workdir})
	if err != nil {
		k.cannot("the JVM could not be started: " + err.Error())
	}
	pid := strconv.Itoa(n)
	if writeText(filepath.Join(workdir, "pid"), pid+"\n") != nil {
		k.cannot("cannot record the JVM's pid")
	}
	// ⛔ THE CORE FIRST, THEN THE TORRENT. Measured on 2026-10-21: a torrent on
	// the first launch's command line reaches the console UI before the core
	// has started, the UI asks the core for its download manager, and Main
	// ends the JVM with *Start fails: Core not running*. See bbCoreState.
	started := false
	for waited := 0; waited < 90 && !started; waited++ {
		switch bbCoreState(session()) {
		case "started":
			started = true
			continue
		case "failed":
			k.refuse("the core did not start; client.out has its words")
		}
		if !alive(pid) {
			k.refuse("the JVM exited before the core started; client.out has its words")
		}
		time.Sleep(time.Second)
	}
	if !started {
		k.refuse("the core did not start within 90s")
	}
	// ⭐ THE TORRENT IS HANDED OVER THE WAY THE VENDOR HANDS ONE TO A RUNNING
	// INSTANCE: the same launch with the torrent as its argument, which finds
	// the instance port held, sends its arguments there and exits. The core's
	// StartServer opens it.
	hand := exec.Command("timeout", append([]string{"60", java}, bbLaunchArgs(dir, ":", config, instance, major, nil, "--ui=console", torrent)...)...)
	hand.Env = env
	if rc := run(hand, stdio{in: os.DevNull, out: filepath.Join(workdir, "handover.out"), outAppend: true, err: "&1", dir: workdir}); rc != 0 {
		k.refuse(fmt.Sprintf("the torrent could not be handed to the running instance (exit %d); handover.out has its words", rc))
	}
	for waited := 0; waited < 30; waited++ {
		if bbOpened(session(), torrent) {
			fmt.Printf("started BiglyBT %s pid %s on port %s, Java %s\n", a.version, pid, port, a.java)
			return
		}
		if !alive(pid) {
			k.refuse("the JVM exited after the torrent was handed over; client.out has its words")
		}
		time.Sleep(time.Second)
	}
	k.refuse("the running instance never opened the torrent")
}

func bbStop(k *adapterKit, args []string) {
	if len(args) < 1 {
		k.cannot("stop takes <workdir>")
	}
	workdir := args[0]
	// ⛔ THE KEYS LEAVE THE WORKDIR ON EVERY PATH OUT OF HERE, once the JVM has
	// gone and can write nothing more. See bbScrub.
	defer func() {
		gone, err := bbScrub(workdir, filepath.Join(workdir, "config"))
		if err != nil {
			k.say("the workdir could not be scrubbed: " + err.Error())
			return
		}
		if len(gone) > 0 {
			k.say(fmt.Sprintf("removed %d file(s) of the client's config and keys", len(gone)))
		}
	}()
	pid := readPID(filepath.Join(workdir, "pid"))
	if pid == "" {
		return
	}
	n, _ := strconv.Atoi(pid)
	how := "already exited"
	// ⚠ SIGTERM RUNS THE CORE'S SHUTDOWN HOOK, which stops every download and
	// so announces `stopped`. Twenty seconds, because the core's own tracker
	// closedown timeout is twenty, then KILL.
	if signalPID(pid, syscall.SIGTERM) {
		how = "sigterm"
		if !waitGone(pid, 20) {
			signalPID(pid, syscall.SIGKILL)
			how = "killed after a sigterm it did not finish"
			time.Sleep(time.Second)
		}
	}
	_ = appendText(filepath.Join(workdir, "stop.txt"), bbStopRecord(utcNow(), n, how))
}

//go:build windows

// adapter_qbittorrent-enhanced_windows - the CLIENT-02 capture adapter,
// Windows half, run as `bit-check adapter qbittorrent-enhanced <subcommand>`.
//
// The contract is scripts/capture/adapters/README.md and the Linux half is
// adapter_qbittorrent-enhanced.go. What this decides without Windows in it is
// qbtenhanced.go, which any host asks through `bit-check qbittorrent-enhanced
// --selftest`.
//
// ⚠ THE MECHANICS ARE THE qBittorrent AND aria2-next WINDOWS HALVES', REUSED
// AND NOT COPIED: qbtWinRun, qbtWin7z, qbtWinFindPayload, qbtWinLaunch,
// qbtWinWaiter, qbtWinAPI, qbtWinShutdown, qbtWinFreePort, the throwaway hash
// and its scrub, and aria2-next's Scoop bootstrap. Every DECISION here - the
// scheme, the asset, the index, the profile, the version shape - is
// Enhanced's own, in qbtenhanced.go.
//
// -- ⛔ WHAT DIFFERS FROM LINUX, AND WHY EACH IS A FACT OF THE BUILD ---------
//
//   - The vendor ships no nox for Windows: the installer's one executable is the
//     GUI `qbittorrent.exe`, and `--version` is compiled out of it exactly as in
//     upstream (`cmdoptions.cpp` is byte-identical to `release-5.2.4`'s). So
//     `version` starts the build in a throwaway profile with the Web UI on a
//     free loopback port and asks `/api/v2/app/version`, keeping
//     `/api/v2/app/buildInfo` - which names libtorrent - on stderr.
//   - `stop` asks `/api/v2/app/shutdown`, and kills a build that has not exited
//     ten seconds later; `stop.txt` says which ended it.
//
// -- the routes, measured on 2026-10-21 -------------------------------------------
//
// ⛔ `release`: `qbittorrent_enhanced_<version>_x64_setup.exe` from GitHub,
// fetched by `install-rootless --fetch` under the listing's digest and
// EXTRACTED by the runner's 7-Zip into `<prefix>\opt\qbittorrent-enhanced`. No
// installer code runs.
// ⛔ `package`: Scoop's `extras/qbittorrent-enhanced`, the one package manager
// at 5.2.4.10 (winget and Chocolatey stop at 5.2.3.10). Scoop is installed by
// its own installer into `%USERPROFILE%\scoop`, the `extras` bucket is added
// at whatever commit is current, and `scoop install` fetches the installer,
// checks it against the manifest's hash and extracts it with 7-Zip into its
// own version directory. No version is passed: the bucket is the resolver.
// ⚠ THE PAIR SHARES ITS ORIGIN: Scoop's URL is the release route's own GitHub
// asset. Resolver (Scoop's bucket against the listing), delivery and check
// (Scoop's downloader against install-rootless) differ, which is what
// `E-ACQ-07` and `E-ACQ-08` compare, as for aria2-next's pair.
//
// Exit codes: the contract's, 0 done, 1 refused, 2 could not run.
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

func init() { registerAdapter(qbeTarget, qbeWinAdapter) }

func qbeWinAdapter(sub string, args []string) int {
	var err error
	switch sub {
	case "describe":
		err = qbeWinDescribe()
	case "install":
		if len(args) < 2 {
			err = qbtCannot("install takes <route> <workdir>")
		} else {
			err = qbeWinInstall(args[0], args[1])
		}
	case "version":
		err = qbeWinVersion()
	case "start":
		if len(args) < 3 {
			err = qbtCannot("start takes <torrent> <workdir> <peer-port>")
		} else {
			err = qbeWinStart(args[0], args[1], args[2])
		}
	case "stop":
		if len(args) < 1 {
			err = qbtCannot("stop takes <workdir>")
		} else {
			err = qbeWinStop(args[0])
		}
	default:
		err = qbtCannot("unknown subcommand: %s", sub)
	}
	if err == nil {
		return 0
	}
	if e, ok := err.(qbtWinExit); ok {
		fmt.Fprintf(os.Stderr, "%s: %s\n", qbeTarget, e.msg)
		return e.code
	}
	fmt.Fprintf(os.Stderr, "%s: %v\n", qbeTarget, err)
	return 2
}

// -- where the build is ---------------------------------------------------------

// qbeWinReleaseDir is where the release route places the payload. ⛔ THE PREFIX
// IS ASKED FOR, NEVER COMPOSED.
func qbeWinReleaseDir() (string, error) {
	prefix, err := rootlessPrefix()
	if err != nil {
		return "", err
	}
	return filepath.Join(prefix, "opt", qbeTarget), nil
}

func qbeWinScoopAppDir(root string) string { return filepath.Join(root, "apps", qbeScoopApp) }

// qbeWinBinary is the installed executable, or "". ⭐ The release prefix first,
// then Scoop's version directory. ⛔ No PATH lookup: Scoop's shim is Scoop's
// program, and a parent `qbittorrent.exe` on PATH is another product.
func qbeWinBinary() string {
	if v := os.Getenv("BIT_IDS_QBITTORRENT_ENHANCED"); v != "" {
		return v
	}
	if dir, err := qbeWinReleaseDir(); err == nil {
		if exe, err := qbtWinFindPayload(dir); err == nil {
			return exe
		}
	}
	if root := a2nWinScoopDir(); root != "" {
		if _, exe, err := qbeScoopBuild(qbeWinScoopAppDir(root)); err == nil {
			return exe
		}
	}
	return ""
}

// -- describe -------------------------------------------------------------------

func qbeWinDescribe() error {
	var b strings.Builder
	fmt.Fprintf(&b, "target=%s\n", qbeTarget)
	b.WriteString("kind=stock\n")
	if exe := qbeWinBinary(); exe != "" {
		fmt.Fprintf(&b, "binary=%s\n", exe)
	}
	// ⛔ THE LINUX HALF'S SCHEME, because it is the same repository's releases;
	// only the asset differs. ⭐ `{version}` names one asset of 13.
	b.WriteString(qbeDescribeScheme(qbeWindowsAsset))
	// ⚠ Neither route compiles, so there is no `compiles` key.
	b.WriteString("routes=package,release\n")
	fmt.Print(b.String())
	return nil
}

// -- install --------------------------------------------------------------------

func qbeWinInstall(route, workdir string) error {
	if err := os.MkdirAll(workdir, 0o755); err != nil {
		return qbtCannot("cannot create %s", workdir)
	}
	format, ok := qbePackageFormat(true, route)
	if !ok {
		return qbtCannot("this adapter does not say how the %s route packages a build", route)
	}
	log := filepath.Join(workdir, "install.log")
	var err error
	switch route {
	case "release":
		err = qbeWinInstallRelease(workdir, log)
	case "package":
		err = qbeWinInstallPackage(workdir, log)
	}
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(workdir, "package"), []byte(format+"\n"), 0o644); err != nil {
		return qbtRefuse("the route installed and could not record how it packaged the build")
	}
	return nil
}

func qbeWinInstallRelease(workdir, log string) error {
	url := os.Getenv("BIT_IDS_RELEASE_URL")
	if url == "" {
		return qbtCannot("the release route needs BIT_IDS_RELEASE_URL, resolved before the route was cut")
	}
	sevenZip := qbtWin7z()
	if sevenZip == "" {
		return qbtCannot("the release route extracts with 7-Zip and neither %%ProgramFiles%%\\7-Zip\\7z.exe nor 7z is on this host")
	}
	self, err := os.Executable()
	if err != nil {
		return qbtCannot("cannot find this binary to run install-rootless with")
	}
	disposition := []string{"--digest-unpublished", qbeDigestUnpublished}
	if res := os.Getenv("BIT_IDS_RELEASE_RESOLUTION"); res != "" {
		disposition = []string{"--from-resolution", res}
	}
	setup := filepath.Join(workdir, "qbittorrent-enhanced-setup.exe")
	args := append([]string{"install-rootless", "--fetch", "--url", url, "--into", setup}, disposition...)
	args = append(args, "--report", filepath.Join(workdir, "rootless-install.txt"))
	switch rc := qbtWinRun(log, self, args...); rc {
	case 0:
	case 1:
		return qbtRefuse("the release route could not fetch the vendor's installer")
	default:
		return qbtCannot("the rootless installer could not run for the release route (exit %d)", rc)
	}
	// ⛔ EXTRACTED, NEVER RUN, into a staging directory renamed into place only
	// once exactly one `qbittorrent.exe` is found in it.
	dir, err := qbeWinReleaseDir()
	if err != nil {
		return qbtCannot("the rootless prefix could not be derived")
	}
	staged := dir + ".new"
	if err := os.RemoveAll(staged); err != nil {
		return qbtRefuse("cannot clear %s", staged)
	}
	if err := os.MkdirAll(filepath.Dir(dir), 0o755); err != nil {
		return qbtRefuse("this user cannot create %s; a rootless install writes nowhere else", filepath.Dir(dir))
	}
	if rc := qbtWinRun(log, sevenZip, "x", "-y", "-bd", "-o"+staged, setup); rc != 0 {
		return qbtRefuse("7-Zip could not extract the installer (exit %d)", rc)
	}
	if _, err := qbtWinFindPayload(staged); err != nil {
		return qbtRefuse("the extracted installer carries no single build: %v", err)
	}
	if err := os.RemoveAll(dir); err != nil {
		return qbtRefuse("cannot clear %s", dir)
	}
	if err := os.Rename(staged, dir); err != nil {
		return qbtRefuse("cannot place the payload at %s", dir)
	}
	sum, _ := fileSHA256(sevenZip)
	exe, _ := qbtWinFindPayload(dir)
	exeSum, _ := fileSHA256(exe)
	note := fmt.Sprintf("extractor=%s\nextractor_sha256=%s\npayload=%s\nexecutable=%s\nexecutable_sha256=%s\n",
		sevenZip, sum, dir, exe, exeSum)
	if err := os.WriteFile(filepath.Join(workdir, "extract.txt"), []byte(note), 0o644); err != nil {
		return qbtRefuse("the route could not record how it extracted the installer")
	}
	return nil
}

func qbeWinInstallPackage(workdir, log string) error {
	pwsh, err := exec.LookPath("pwsh")
	if err != nil {
		return qbtCannot("the package route runs Scoop under pwsh and it is not on this host")
	}
	git, err := exec.LookPath("git")
	if err != nil {
		return qbtCannot("the package route adds Scoop's extras bucket with git and git is not on this host")
	}
	root := a2nWinScoopDir()
	if root == "" {
		return qbtCannot("there is no absolute home directory to install Scoop under")
	}
	// ⛔ A SCOOP ALREADY HERE IS REFUSED: its buckets would be at a state this
	// route did not read.
	if entries, err := os.ReadDir(root); err == nil && len(entries) > 0 {
		return qbtRefuse("a Scoop is already at %s, so its index state is not this route's", root)
	}
	self, err := os.Executable()
	if err != nil {
		return qbtCannot("cannot find this binary to run install-rootless with")
	}
	installer := filepath.Join(workdir, "scoop-install.ps1")
	switch rc := qbtWinRun(log, self, "install-rootless", "--fetch", "--url", a2nWinScoopInstaller,
		"--into", installer, "--digest-unpublished", a2nWinScoopInstallerUnpublished,
		"--report", filepath.Join(workdir, "scoop-installer.txt")); rc {
	case 0:
	case 1:
		return qbtRefuse("the package route could not fetch Scoop's installer")
	default:
		return qbtCannot("the rootless installer could not run for Scoop's installer (exit %d)", rc)
	}
	// ⚠ `-RunAsAdmin` because a hosted runner's user is an administrator, which
	// Scoop's installer refuses by default; it changes nothing else.
	os.Setenv("SCOOP", root)
	if rc := a2nWinPwsh(log, pwsh, installer, "-ScoopDir", root, "-RunAsAdmin"); rc != 0 {
		return qbtRefuse("Scoop's installer did not install Scoop into %s (exit %d)", root, rc)
	}
	scoop := filepath.Join(root, "apps", "scoop", "current", "bin", "scoop.ps1")
	if !isRegular(scoop) {
		return qbtRefuse("Scoop's installer left no %s", scoop)
	}
	// ⭐ THE extras BUCKET IS SCOOP'S OWN KNOWN BUCKET, added by name: Scoop
	// clones it from the URL its own `buckets.json` names.
	if rc := a2nWinPwsh(log, pwsh, scoop, "bucket", "add", qbeScoopBucket); rc != 0 {
		return qbtRefuse("Scoop could not add its %s bucket (exit %d)", qbeScoopBucket, rc)
	}
	bucket := filepath.Join(root, "buckets", qbeScoopBucket)
	manifest, err := os.ReadFile(filepath.Join(bucket, "bucket", qbeScoopApp+".json"))
	if err != nil {
		return qbtRefuse("Scoop's %s bucket carries no %s.json", qbeScoopBucket, qbeScoopApp)
	}
	_ = os.WriteFile(filepath.Join(workdir, "scoop-manifest.json"), manifest, 0o644)
	entry, err := qbeScoopManifest(manifest)
	if err != nil {
		return qbtRefuse("%v", err)
	}
	out, err := exec.Command(git, "-C", bucket, "rev-parse", "HEAD").Output()
	commit := strings.TrimSpace(string(out))
	if err != nil || !lowerHex40(commit) {
		return qbtRefuse("Scoop's %s bucket at %s has no git commit to name its state by", qbeScoopBucket, bucket)
	}
	note := fmt.Sprintf("index=scoop\nbucket=%s\nbucket_commit=%s\napp=%s\nindex_version=%s\nindex_url=%s\nindex_sha256=%s\n",
		qbeScoopBucket, commit, qbeScoopApp, entry.version, entry.url, entry.hash)
	if err := os.WriteFile(filepath.Join(workdir, "scoop-index.txt"), []byte(note), 0o644); err != nil {
		return qbtRefuse("the route could not record what the index named")
	}
	// ⛔ NO VERSION; `--no-update-scoop` keeps the bucket at the commit just read.
	// ⚠ Scoop installs its own 7-Zip from its `main` bucket to extract the
	// installer, which is Scoop's behaviour and not this route's choice.
	if rc := a2nWinPwsh(log, pwsh, scoop, "install", qbeScoopBucket+"/"+qbeScoopApp, "--no-update-scoop"); rc != 0 {
		return qbtRefuse("the package route did not install %s/%s (scoop exit %d)", qbeScoopBucket, qbeScoopApp, rc)
	}
	placed, exe, err := qbeScoopBuild(qbeWinScoopAppDir(root))
	if err != nil {
		return qbtRefuse("%v", err)
	}
	if placed != entry.version {
		return qbtRefuse("Scoop's manifest named %s and Scoop placed %s", entry.version, placed)
	}
	sum, _ := fileSHA256(exe)
	appendFile(filepath.Join(workdir, "scoop-index.txt"), fmt.Sprintf("placed=%s\nplaced_sha256=%s\n", exe, sum))
	if err := writePackageIndex(workdir, "scoop", qbeScoopBucket+"/"+qbeScoopApp, entry.version, entry.url); err != nil {
		return qbtRefuse("the route could not record what the index named")
	}
	return nil
}

// -- version --------------------------------------------------------------------

// qbeWinVersion asks the installed build over its Web UI. ⛔ THE BUILD SPEAKING:
// `/api/v2/app/version` is `QBT_VERSION` compiled into the executable. ⚠ IT
// STARTS THE BUILD, in a profile made for this call and deleted after it, with
// every surface off and no torrent, and ends it on every path out.
func qbeWinVersion() error {
	exe := qbeWinBinary()
	if exe == "" {
		return qbtCannot("%s is not installed", qbtWinExe)
	}
	tmp, err := os.MkdirTemp("", "qbe-version-")
	if err != nil {
		return qbtCannot("cannot make a temporary profile")
	}
	defer os.RemoveAll(tmp)
	port, err := qbtWinFreePort()
	if err != nil {
		return qbtCannot("no loopback port is free to ask the build on")
	}
	hash, err := qbtWinRandomHash()
	if err != nil {
		return qbtCannot("cannot draw the Web UI's throwaway credential")
	}
	config := filepath.Join(tmp, "profile", "qBittorrent", "config")
	if err := os.MkdirAll(config, 0o755); err != nil {
		return qbtCannot("cannot create the temporary profile")
	}
	text := qbeProfile(filepath.Join(tmp, "downloads"), "0", port, hash)
	if msg := qbeProfileRefusal(text); msg != "" {
		return qbtRefuse("the profile would start the build with %s", msg)
	}
	if err := os.WriteFile(filepath.Join(config, "qBittorrent.ini"), []byte(text), 0o644); err != nil {
		return qbtCannot("the temporary profile could not be written")
	}
	out := filepath.Join(tmp, "client.out")
	cmd, err := qbtWinLaunch(out, exe, "--profile="+filepath.Join(tmp, "profile"), "--no-splash")
	if err != nil {
		return qbtCannot("%s could not be started: %v", exe, err)
	}
	exited := qbtWinWaiter(cmd.Process)
	wait := time.Duration(envSeconds("BIT_IDS_QBT_ANSWER_TIMEOUT", 30)) * time.Second
	deadline := time.Now().Add(wait)
	var answer, info []byte
	for time.Now().Before(deadline) {
		if code, body, err := qbtWinAPI(port, http.MethodGet, "/api/v2/app/version"); err == nil && code == http.StatusOK {
			answer = body
			break
		}
		if exited(500 * time.Millisecond) {
			break
		}
	}
	if answer != nil {
		if code, body, err := qbtWinAPI(port, http.MethodGet, "/api/v2/app/buildInfo"); err == nil && code == http.StatusOK {
			info = body
		}
	}
	how := qbtWinShutdown(cmd.Process, exited, port, 10*time.Second)
	if answer == nil {
		if said, _ := os.ReadFile(out); len(bytes.TrimSpace(said)) > 0 {
			fmt.Fprintf(os.Stderr, "%s: the build printed:\n%s\n", qbeTarget, bytes.TrimSpace(said))
		}
		return qbtCannot("%s did not answer on its Web UI within %s (%s)", exe, wait, how)
	}
	version, err := qbtWinParseVersion(string(answer))
	if err != nil {
		return qbtCannot("%v", err)
	}
	// ⛔ ENHANCED'S SHAPE: four components. A three-component answer is the
	// parent's build, whatever file it was found in.
	if !qbeVersionRe.MatchString(version) {
		return qbtCannot("the build answered %s, which is not an Enhanced four-component version", version)
	}
	fmt.Fprintf(os.Stderr, "%s reported, asked over /api/v2/app/version on 127.0.0.1:%d (%s):\n%s\n",
		qbeTarget, port, how, strings.TrimSpace(string(answer)))
	if info != nil {
		var pretty bytes.Buffer
		if json.Indent(&pretty, info, "", "  ") == nil {
			info = pretty.Bytes()
		}
		fmt.Fprintf(os.Stderr, "/api/v2/app/buildInfo:\n%s\n", info)
	} else {
		fmt.Fprintln(os.Stderr, "/api/v2/app/buildInfo: no answer")
	}
	fmt.Println(version)
	return nil
}

// -- start and stop ---------------------------------------------------------------

func qbeWinStart(torrent, workdir, peerPort string) error {
	exe := qbeWinBinary()
	if exe == "" {
		return qbtCannot("%s is not installed", qbtWinExe)
	}
	if fi, err := os.Stat(torrent); err != nil || fi.Size() == 0 {
		return qbtCannot("there is no torrent at %s", torrent)
	}
	profile := filepath.Join(workdir, "profile")
	config := filepath.Join(profile, "qBittorrent", "config")
	downloads := filepath.Join(workdir, "downloads")
	if err := os.MkdirAll(config, 0o755); err != nil {
		return qbtCannot("cannot create the profile under %s", workdir)
	}
	if err := os.MkdirAll(downloads, 0o755); err != nil {
		return qbtCannot("cannot create %s", downloads)
	}
	port, err := qbtWinFreePort()
	if err != nil {
		return qbtCannot("no loopback port is free for the Web UI that stops the build")
	}
	hash, err := qbtWinRandomHash()
	if err != nil {
		return qbtCannot("cannot draw the Web UI's throwaway credential")
	}
	text := qbeProfile(downloads, peerPort, port, hash)
	if msg := qbeProfileRefusal(text); msg != "" {
		return qbtRefuse("the profile would start the build with %s", msg)
	}
	if err := os.WriteFile(filepath.Join(config, "qBittorrent.ini"), []byte(text), 0o644); err != nil {
		return qbtCannot("the profile could not be written")
	}
	if err := os.WriteFile(filepath.Join(workdir, "webui-port"), []byte(strconv.Itoa(port)+"\n"), 0o644); err != nil {
		return qbtCannot("cannot record the Web UI port stop will use")
	}
	// ⛔ THE TORRENT IS A POSITIONAL ARGUMENT, as on Linux.
	cmd, err := qbtWinLaunch(filepath.Join(workdir, "client.out"), exe,
		"--profile="+profile, "--relative-fastresume", "--skip-dialog=true", "--no-splash", torrent)
	if err != nil {
		return qbtCannot("%s could not be started: %v", exe, err)
	}
	pid := cmd.Process.Pid
	if err := os.WriteFile(filepath.Join(workdir, "pid"), []byte(strconv.Itoa(pid)+"\n"), 0o644); err != nil {
		return qbtCannot("cannot record the build's process id")
	}
	cmd.Process.Release()
	fmt.Printf("started %s pid %d on port %s, Web UI 127.0.0.1:%d\n", qbtWinExe, pid, peerPort, port)
	return nil
}

func qbeWinStop(workdir string) error {
	// ⛔ THE CREDENTIAL LEAVES THE PROFILE ON EVERY PATH OUT OF HERE.
	defer func() {
		if n, err := qbtWinScrub(filepath.Join(workdir, "profile")); err != nil {
			fmt.Fprintf(os.Stderr, "%s: the profile could not be scrubbed: %v\n", qbeTarget, err)
		} else if n > 0 {
			fmt.Fprintf(os.Stderr, "%s: removed %d credential line(s) from the profile\n", qbeTarget, n)
		}
	}()
	pid, err := strconv.Atoi(strings.TrimSpace(readTrimmed(filepath.Join(workdir, "pid"))))
	if err != nil || pid <= 1 {
		return nil
	}
	p, err := os.FindProcess(pid)
	if err != nil {
		return nil
	}
	port, _ := strconv.Atoi(strings.TrimSpace(readTrimmed(filepath.Join(workdir, "webui-port"))))
	how := qbtWinShutdown(p, qbtWinWaiter(p), port, 10*time.Second)
	appendFile(filepath.Join(workdir, "stop.txt"), fmt.Sprintf("%s pid %d: %s\n", utcNow(), pid, how))
	fmt.Printf("stopped %s pid %d: %s\n", qbtWinExe, pid, how)
	return nil
}

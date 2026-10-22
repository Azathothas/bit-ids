// biglybt - what the `biglybt` adapter decides, on either platform, as
// functions with no process in them, so any host can ask them.
//
// CLIENT-09. The adapter is two files that build for one platform each,
// adapter_biglybt.go off Windows and adapter_biglybt_windows.go on it;
// everything here is the part of them a Linux gate can prove: the version
// scheme, the settings document the client reads, its refusal of a document
// with a surface on, the class that asks the installed build its version, how
// that answer is read, what the source build compiles, the vendor installer's
// command line, and the scrub of the keys the client writes. `bit-check
// biglybt --selftest` asks each of them.
//
// -- the vendor, measured on 2026-10-21 ---------------------------------------
//
// BiglyBT's newest stable is 4.1.0.0, released on 2026-05-20 as the GitHub
// release `v4.1.0.0`. It is Java, built for Java 8 (class version 52), and runs
// on any JVM from 1.8 to 25. The release carries a plain Linux tarball,
// `GitHub_BiglyBT_unix.tar.gz` (the jar, `commons-cli.jar`, the vendor's
// launcher script, eleven bundled plugins and SWT, no JRE), install4j
// installers for Linux and Windows, the jar alone, and `GitHub_sha256sum.txt`,
// which names the tarball and the installers and not the jar.
//
//	platform  route    what it installs
//	Linux     release  the vendor's tarball, extracted into the prefix
//	Linux     source   the `v4.1.0.0` tag, compiled with the host's JDK
//	Windows   release  the vendor's install4j installer, with its bundled JRE
//	Windows   package  winget's `BiglySoftware.BiglyBT`: the vendor's NoJRE
//	                   installer and Temurin's JRE as its dependency
//
// ⛔ Both Linux routes run on the host's JVM, so they run one runtime. The two
// Windows routes run the vendor's bundled JRE and winget's Temurin, which is
// the packaging difference this entry exists to record rather than flatten.
//
// Exit codes: the selftest's, 0 every case held, 1 one did not, 2 could not run.
package main

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

const (
	bbTarget   = "biglybt"
	bbRepo     = "BiglySoftware/BiglyBT"
	bbWingetID = "BiglySoftware.BiglyBT"
	bbJar      = "BiglyBT.jar"
	bbCLIJar   = "commons-cli.jar"
	bbMain     = "com.biglybt.ui.Main"
	// bbProbeClass is the class bbVersionClass makes, and the name it is run by.
	bbProbeClass = "BitIdsBiglyVersion"
	// bbLinuxAsset and bbWindowsAsset are each half's release artifact. ⚠ The
	// vendor's prefix moves between `GitHub_` and `Github_` from release to
	// release, so the pattern starts at the product's own name.
	bbLinuxAsset   = "*BiglyBT_unix.tar.gz"
	bbWindowsAsset = "*BiglyBT_Windows_Installer.exe"
	// bbDigestAsset is the vendor's checksums document: `GitHub_sha256sum.txt`
	// in 4.1.0.0, `Github_sha256sum.txt` in 4.0.0.0.
	bbDigestAsset = "*sha256sum.txt"
)

// bbDescribeScheme is what both halves print for `resolve-release` and
// `resolve-source`.
//
// ⛔ `v`, EXACTLY FOUR COMPONENTS, FLOOR 2. The repository's 34 tags on
// 2026-10-21 are `v1.0.0.0` to `v4.1.0.0`, four components each, except
// `v1.8.00`, three, under major 1: below the floor it is released rather than
// blocking. Every release object is one of those tags.
func bbDescribeScheme(asset string) string {
	return "release_repo=" + bbRepo + "\n" +
		"release_tag_prefix=v\n" +
		"release_min_components=4\n" +
		"release_max_components=4\n" +
		"release_floor_major=2\n" +
		"release_series=-\n" +
		"release_asset=" + asset + "\n" +
		"release_digest_asset=" + bbDigestAsset + "\n"
}

// bbPackageFormat is what each route delivers, as `Build.package` spells it.
func bbPackageFormat(windows bool, route string) (string, bool) {
	switch {
	case !windows && route == "release":
		return "java-tarball", true
	case !windows && route == "source":
		return "java-source", true
	case windows && route == "release":
		return "install4j-installer", true
	case windows && route == "package":
		return "install4j-installer-nojre", true
	}
	return "", false
}

// bbTagVersion is the version a `v` tag names, or "".
func bbTagVersion(tag string) string {
	v, ok := strings.CutPrefix(tag, "v")
	if !ok || !bbVersionRe.MatchString(v) {
		return ""
	}
	return v
}

var bbVersionRe = regexp.MustCompile(`^[0-9]+\.[0-9]+\.[0-9]+\.[0-9]+$`)

// bbSameVersion says whether an index's version and the build's name one
// release. ⚠ winget's index spells 4.1.0.0 as `4.1`, so trailing zero
// components are not a difference; any other is.
func bbSameVersion(index, built string) bool {
	norm := func(v string) (string, bool) {
		parts := strings.Split(v, ".")
		for _, p := range parts {
			if p == "" || strings.Trim(p, "0123456789") != "" {
				return "", false
			}
		}
		for len(parts) > 1 && strings.Trim(parts[len(parts)-1], "0") == "" {
			parts = parts[:len(parts)-1]
		}
		for i, p := range parts {
			n, _ := strconv.Atoi(p)
			parts[i] = strconv.Itoa(n)
		}
		return strings.Join(parts, "."), true
	}
	a, ok1 := norm(index)
	b, ok2 := norm(built)
	return ok1 && ok2 && a == b
}

// -- the settings -------------------------------------------------------------

// bbSetting is one key of the client's config file, `biglybt.config`, a
// bencoded dictionary. An int is a bencoded integer, which is how the client
// stores a boolean.
type bbSetting struct {
	key string
	val any
}

// bbBundledPlugins are the eleven plugins the vendor's tarball carries in
// `plugins/`, read on 2026-10-21. ⛔ None is loaded, on either route.
//
//   - mlDHT is the mainline DHT, azmsgsync and azrating ride the DHT, aercm is
//     Swarm Discoveries over it, mlab is a speed test against M-Lab, azupnpav
//     is a UPnP media server with SSDP, and azupdater and dorkboxupdater update
//     the client. Each is a network surface this capture switches off.
//   - azutp is uTP. ⚠ The source route builds the client's tag, which carries
//     none of these plugins, so loading them on the release route alone would
//     make the two routes two programs. So uTP is off on both and the record
//     says uTP was not measured.
//   - azlocprov (GeoIP) and migratetorrentapp have no network surface and go
//     for the same reason.
var bbBundledPlugins = []string{"aercm", "azlocprov", "azmsgsync", "azrating", "azupdater", "azupnpav",
	"azutp", "dorkboxupdater", "migratetorrentapp", "mldht", "mlab"}

// bbSettings is what every session writes, whole, before the client starts.
//
// ⛔ ONE ROW PER LINE AND EVERY SURFACE AT 0. `check-bitcheck` plants the DHT
// row at 1 and the selftest must refuse the document. The keys were read from
// the build's own `set` listing on 2026-10-21, which prints every parameter a
// running 4.1.0.0 knows with its default, and the sockets of a running build
// were read before and after: the vendor's defaults open UDP and TCP on a
// random data port, mlDHT's UDP 49001, the media server's TCP port, and join
// the SSDP and instance multicast groups; with these rows only the peer port
// and two loopback listeners remain.
var bbSettings = []bbSetting{
	{"Plugin.DHT.dht.enabled", 0},
	{"Plugin.mlDHT.enable", 0},
	{"Peer Source Selection Default.DHT", 0},
	{"Peer Source Selection Default.PeerExchange", 0},
	{"Peer Source Selection Default.Plugin", 0},
	{"Plugin.azlocaltracker.Plugin.localtracker.enable", 0},
	{"Plugin.UPnP.upnp.enable", 0},
	{"Plugin.UPnP.natpmp.enable", 0},
	{"update.start", 0},
	{"update.periodic", 0},
	{"update.opendialog", 0},
	{"update.autodownload", 0},
	{"Plugin.PluginUpdate.enable.update", 0},
	{"Send Version Info", 0},
	{"Network Selection Default.I2P", 0},
	{"Network Selection Default.Tor", 0},
	{"Tracker Network Selection Default.I2P", 0},
	{"Tracker Network Selection Default.Tor", 0},
	{"Plugin.azbuddy.azbuddy.enabled", 0},
	{"Plugin.azbuddy.azbuddy.dchat.decentralized.enabled", 0},
	{"Plugin.azbuddy.azbuddy.tracker.enabled", 0},
	{"rcm.overall.enabled", 0},
}

// bbSurfaces are the keys a document must hold at 0, asked of the document
// itself rather than of bbSettings, so a row changed there is caught here.
var bbSurfaces = []string{
	"Plugin.DHT.dht.enabled", "Plugin.mlDHT.enable",
	"Peer Source Selection Default.DHT", "Peer Source Selection Default.PeerExchange",
	"Plugin.azlocaltracker.Plugin.localtracker.enable",
	"Plugin.UPnP.upnp.enable", "Plugin.UPnP.natpmp.enable",
	"update.start", "update.periodic", "Plugin.PluginUpdate.enable.update", "Send Version Info",
	"Network Selection Default.I2P", "Network Selection Default.Tor",
	"Tracker Network Selection Default.I2P", "Tracker Network Selection Default.Tor",
	"Plugin.azbuddy.azbuddy.dchat.decentralized.enabled",
}

// bbJVMProperties are the system properties every launch passes beside the
// vendor's own.
//
// ⛔ `az.instance.manager.enable=0` IS THE LAN INSTANCE DISCOVERY, multicast to
// 239.255.67.250:16680, which shares peers between BiglyBT instances on a LAN.
// It has no config key, only this property, read in
// `ClientInstanceManagerImpl`.
var bbJVMProperties = []string{"-Daz.instance.manager.enable=0"}

// bbConfig is the bencoded `biglybt.config` a session starts on: every row of
// bbSettings, every bundled plugin unloaded, the save path, and the peer port
// as the TCP, UDP and non-data UDP ports. ⚠ Port 0 leaves the port to the
// build, as every adapter's does.
func bbConfig(downloads, peerPort string) ([]byte, error) {
	if downloads == "" {
		return nil, errors.New("the settings need a download directory")
	}
	port := 0
	if peerPort != "0" {
		n, err := strconv.Atoi(peerPort)
		if err != nil || n < 1 || n > 65535 || strconv.Itoa(n) != peerPort {
			return nil, fmt.Errorf("the peer port [%s] is not a port", peerPort)
		}
		port = n
	}
	m := map[string]any{}
	for _, s := range bbSettings {
		m[s.key] = s.val
	}
	for _, p := range bbBundledPlugins {
		m["PluginInfo."+p+".enabled"] = 0
	}
	m["Default save path"] = downloads
	if port > 0 {
		m["TCP.Listen.Port"] = port
		m["UDP.Listen.Port"] = port
		m["UDP.NonData.Listen.Port"] = port
	}
	return bencodeDict(m)
}

// bbConfigRefusal answers "" for a document that holds every surface at 0 and
// every bundled plugin unloaded, or what it found on.
func bbConfigRefusal(doc []byte) string {
	v, err := bdecode(doc)
	if err != nil {
		return "a document that does not decode: " + err.Error()
	}
	m, ok := v.(map[string]any)
	if !ok {
		return "a document that is not a dictionary"
	}
	var on []string
	zero := func(k string) {
		if n, ok := m[k].(int64); !ok || n != 0 {
			on = append(on, k)
		}
	}
	for _, k := range bbSurfaces {
		zero(k)
	}
	for _, p := range bbBundledPlugins {
		zero("PluginInfo." + p + ".enabled")
	}
	if len(on) == 0 {
		return ""
	}
	return "these surfaces not held off: " + strings.Join(on, ", ")
}

// bbSettingsText is the document as `key=value` lines, sorted, which `start`
// keeps in the workdir: the bencoded file itself is scrubbed with the rest of
// the config directory.
func bbSettingsText(doc []byte) string {
	v, err := bdecode(doc)
	m, _ := v.(map[string]any)
	if err != nil || m == nil {
		return ""
	}
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var b strings.Builder
	for _, k := range keys {
		switch x := m[k].(type) {
		case int64:
			fmt.Fprintf(&b, "%s=%d\n", k, x)
		case []byte:
			fmt.Fprintf(&b, "%s=%s\n", k, x)
		}
	}
	return b.String()
}

// -- bencoding ----------------------------------------------------------------

// bencodeDict encodes a dictionary of ints and strings, keys sorted as the
// format requires.
func bencodeDict(m map[string]any) ([]byte, error) {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var b bytes.Buffer
	b.WriteByte('d')
	for _, k := range keys {
		fmt.Fprintf(&b, "%d:%s", len(k), k)
		switch x := m[k].(type) {
		case int:
			fmt.Fprintf(&b, "i%de", x)
		case string:
			fmt.Fprintf(&b, "%d:%s", len(x), x)
		default:
			return nil, fmt.Errorf("the setting %s is neither an int nor a string", k)
		}
	}
	b.WriteByte('e')
	return b.Bytes(), nil
}

// bdecode decodes one whole bencoded value: int64, []byte, []any or
// map[string]any. Trailing bytes are refused.
func bdecode(b []byte) (any, error) {
	v, n, err := bdecodeAt(b, 0, 0)
	if err != nil {
		return nil, err
	}
	if n != len(b) {
		return nil, fmt.Errorf("%d bytes after the value", len(b)-n)
	}
	return v, nil
}

func bdecodeAt(b []byte, i, depth int) (any, int, error) {
	if depth > 64 {
		return nil, i, errors.New("nested too deeply")
	}
	if i >= len(b) {
		return nil, i, errors.New("truncated")
	}
	switch c := b[i]; {
	case c == 'i':
		j := bytes.IndexByte(b[i:], 'e')
		if j < 0 {
			return nil, i, errors.New("an integer with no end")
		}
		n, err := strconv.ParseInt(string(b[i+1:i+j]), 10, 64)
		if err != nil {
			return nil, i, fmt.Errorf("a bad integer at %d", i)
		}
		return n, i + j + 1, nil
	case c == 'l':
		out := []any{}
		i++
		for i < len(b) && b[i] != 'e' {
			v, n, err := bdecodeAt(b, i, depth+1)
			if err != nil {
				return nil, n, err
			}
			out = append(out, v)
			i = n
		}
		if i >= len(b) {
			return nil, i, errors.New("a list with no end")
		}
		return out, i + 1, nil
	case c == 'd':
		out := map[string]any{}
		i++
		for i < len(b) && b[i] != 'e' {
			k, n, err := bdecodeAt(b, i, depth+1)
			if err != nil {
				return nil, n, err
			}
			ks, ok := k.([]byte)
			if !ok {
				return nil, i, fmt.Errorf("a key that is not a string at %d", i)
			}
			v, n2, err := bdecodeAt(b, n, depth+1)
			if err != nil {
				return nil, n2, err
			}
			out[string(ks)] = v
			i = n2
		}
		if i >= len(b) {
			return nil, i, errors.New("a dictionary with no end")
		}
		return out, i + 1, nil
	case c >= '0' && c <= '9':
		j := bytes.IndexByte(b[i:], ':')
		if j < 0 {
			return nil, i, errors.New("a string with no length")
		}
		n, err := strconv.Atoi(string(b[i : i+j]))
		start := i + j + 1
		if err != nil || n < 0 || start+n > len(b) {
			return nil, i, fmt.Errorf("a bad string at %d", i)
		}
		return b[start : start+n], start + n, nil
	}
	return nil, i, fmt.Errorf("an unknown value at %d", i)
}

// -- asking the build ---------------------------------------------------------

// bbProbeLines are what the class prints, in order: the product's own two
// constants, read by reflection from the installed jar, and the JVM's own
// properties. ⛔ REFLECTION, BECAUSE `BIGLYBT_VERSION` IS A COMPILE-TIME
// CONSTANT: a class compiled against it would carry the version it was
// compiled against and not the one installed. Nothing here is compiled
// against BiglyBT at all.
var bbProbeLines = []struct {
	key, class, field string // field "" means a system property named key
	bytes             bool
}{
	{"biglybt.version", "com.biglybt.core.util.Constants", "BIGLYBT_VERSION", false},
	{"biglybt.peer_id_prefix", "com.biglybt.core.util.Constants", "VERSION_ID", true},
	{"java.version", "", "", false},
	{"java.vendor", "", "", false},
	{"java.vm.name", "", "", false},
	{"java.vm.version", "", "", false},
	{"java.runtime.name", "", "", false},
	{"java.runtime.version", "", "", false},
	{"java.home", "", "", false},
	{"os.name", "", "", false},
	{"os.arch", "", "", false},
}

// bbVersionClass is a Java class file, version 52, whose `main` prints
// bbProbeLines as `key=value` lines.
//
// ⛔ MADE HERE, NOT COMPILED AND NOT SHIPPED. The build has no `--version`, and
// a JRE has no compiler, so the class is written byte by byte: one method,
// straight-line code with no branch, so it needs no stack map. Run on the
// route's own JVM with the installed jar on the class path, it is the build's
// own code answering, and it starts nothing: `Constants` reads system
// properties and touches no file.
func bbVersionClass() []byte {
	var pool bytes.Buffer
	count := 1
	index := map[string]int{}
	entry := func(key string, body []byte) int {
		if n, ok := index[key]; ok {
			return n
		}
		pool.Write(body)
		index[key] = count
		count++
		return count - 1
	}
	u2 := func(n int) []byte { return []byte{byte(n >> 8), byte(n)} }
	utf8 := func(s string) int {
		return entry("U"+s, append(append([]byte{1}, u2(len(s))...), s...))
	}
	class := func(name string) int { return entry("C"+name, append([]byte{7}, u2(utf8(name))...)) }
	str := func(s string) int { return entry("S"+s, append([]byte{8}, u2(utf8(s))...)) }
	nat := func(name, desc string) int {
		return entry("N"+name+" "+desc, append(append([]byte{12}, u2(utf8(name))...), u2(utf8(desc))...))
	}
	ref := func(tag byte, cls, name, desc string) int {
		return entry(fmt.Sprintf("R%d %s.%s%s", tag, cls, name, desc),
			append(append([]byte{tag}, u2(class(cls))...), u2(nat(name, desc))...))
	}
	method := func(cls, name, desc string) int { return ref(10, cls, name, desc) }

	this := class(bbProbeClass)
	super := class("java/lang/Object")
	mainName, mainDesc, codeName := utf8("main"), utf8("([Ljava/lang/String;)V"), utf8("Code")
	out := ref(9, "java/lang/System", "out", "Ljava/io/PrintStream;")
	printlnRef := method("java/io/PrintStream", "println", "(Ljava/lang/String;)V")
	concat := method("java/lang/String", "concat", "(Ljava/lang/String;)Ljava/lang/String;")
	valueOf := method("java/lang/String", "valueOf", "(Ljava/lang/Object;)Ljava/lang/String;")
	getProperty := method("java/lang/System", "getProperty", "(Ljava/lang/String;)Ljava/lang/String;")
	forName := method("java/lang/Class", "forName", "(Ljava/lang/String;)Ljava/lang/Class;")
	getField := method("java/lang/Class", "getField", "(Ljava/lang/String;)Ljava/lang/reflect/Field;")
	get := method("java/lang/reflect/Field", "get", "(Ljava/lang/Object;)Ljava/lang/Object;")
	newString := method("java/lang/String", "<init>", "([B)V")
	stringClass, byteArray := class("java/lang/String"), class("[B")

	var code bytes.Buffer
	op := func(b byte, idx ...int) {
		code.WriteByte(b)
		for _, i := range idx {
			code.Write(u2(i))
		}
	}
	const getstatic, ldcW, invokevirtual, invokespecial, invokestatic = 0xb2, 0x13, 0xb6, 0xb7, 0xb8
	const newOp, dup, aconstNull, checkcast, ret = 0xbb, 0x59, 0x01, 0xc0, 0xb1
	for _, l := range bbProbeLines {
		op(getstatic, out)
		op(ldcW, str(l.key+"="))
		switch {
		case l.field == "":
			op(ldcW, str(l.key))
			op(invokestatic, getProperty)
			op(invokestatic, valueOf)
		default:
			if l.bytes {
				op(newOp, stringClass)
				op(dup)
			}
			op(ldcW, str(strings.ReplaceAll(l.class, "/", ".")))
			op(invokestatic, forName)
			op(ldcW, str(l.field))
			op(invokevirtual, getField)
			op(aconstNull)
			op(invokevirtual, get)
			if l.bytes {
				op(checkcast, byteArray)
				op(invokespecial, newString)
			} else {
				op(invokestatic, valueOf)
			}
		}
		op(invokevirtual, concat)
		op(invokevirtual, printlnRef)
	}
	op(ret)

	var b bytes.Buffer
	b.Write([]byte{0xca, 0xfe, 0xba, 0xbe, 0, 0, 0, 52})
	b.Write(u2(count))
	b.Write(pool.Bytes())
	b.Write(u2(0x0021)) // public, super
	b.Write(u2(this))
	b.Write(u2(super))
	b.Write(u2(0)) // interfaces
	b.Write(u2(0)) // fields
	b.Write(u2(1)) // methods
	b.Write(u2(0x0009))
	b.Write(u2(mainName))
	b.Write(u2(mainDesc))
	b.Write(u2(1))
	b.Write(u2(codeName))
	attr := make([]byte, 4)
	binary.BigEndian.PutUint32(attr, uint32(2+2+4+code.Len()+2+2))
	b.Write(attr)
	b.Write(u2(8)) // max stack: out, key, new, dup, class name, field name
	b.Write(u2(1)) // max locals: args
	clen := make([]byte, 4)
	binary.BigEndian.PutUint32(clen, uint32(code.Len()))
	b.Write(clen)
	b.Write(code.Bytes())
	b.Write(u2(0)) // exception table
	b.Write(u2(0)) // code attributes
	b.Write(u2(0)) // class attributes
	return b.Bytes()
}

// bbAnswer is what the probe said, read.
type bbAnswer struct {
	version, peerPrefix, java string
	fields                    map[string]string
}

// bbParseAnswer reads the probe's output. ⛔ The version is four components,
// the peer ID prefix is eight bytes and is the one the version spells, and the
// JVM named its version.
func bbParseAnswer(output string) (bbAnswer, error) {
	a := bbAnswer{fields: map[string]string{}}
	for _, l := range strings.Split(strings.ReplaceAll(output, "\r", ""), "\n") {
		k, v, ok := strings.Cut(l, "=")
		if !ok {
			continue
		}
		if _, dup := a.fields[k]; dup {
			return a, fmt.Errorf("the build's answer names %s twice", k)
		}
		a.fields[k] = v
	}
	a.version, a.peerPrefix, a.java = a.fields["biglybt.version"], a.fields["biglybt.peer_id_prefix"], a.fields["java.version"]
	if !bbVersionRe.MatchString(a.version) {
		return a, fmt.Errorf("the build's answer names no BiglyBT version: [%s]", a.version)
	}
	if len(a.peerPrefix) != 8 || !strings.HasPrefix(a.peerPrefix, "-BI") || a.peerPrefix[7] != '-' {
		return a, fmt.Errorf("the build's answer names no Azureus-style peer ID prefix: [%s]", a.peerPrefix)
	}
	if digits := strings.ReplaceAll(a.version, ".", ""); len(digits) == 4 && a.peerPrefix != "-BI"+digits+"-" {
		return a, fmt.Errorf("the build reports %s and a peer ID prefix of %s, which spells another version", a.version, a.peerPrefix)
	}
	if a.java == "" || a.java == "null" {
		return a, errors.New("the build's answer names no Java version")
	}
	return a, nil
}

// bbJavaMajor is the feature release a `java.version` names: 8 for `1.8.0_x`,
// 17 for `17.0.x`.
func bbJavaMajor(v string) int {
	if rest, ok := strings.CutPrefix(v, "1."); ok {
		v = rest
	}
	end := strings.IndexFunc(v, func(r rune) bool { return r < '0' || r > '9' })
	if end < 0 {
		end = len(v)
	}
	n, _ := strconv.Atoi(v[:end])
	return n
}

// bbLaunchArgs is the JVM command line a session runs, after the java
// executable.
//
// ⭐ THE VENDOR'S OWN LAUNCH, read from its Linux script `biglybt` (version 12):
// `-Xmx256m`, the class path, `java.library.path` and `azureus.install.path`
// at the install, `--add-opens java.base/java.net=ALL-UNNAMED` on Java 11 and
// later, then `com.biglybt.ui.Main`. On Windows the vendor's launcher reads
// its `.vmoptions` file, and vmopts carries those lines first. ⚠ Left out is
// the script's restart and update plumbing, `azureus.script`, which only the
// updater reads and which is off.
// ⛔ AND THE CAPTURE'S OWN: the config directory in the workdir, the
// instance port drawn for this session, bbJVMProperties, the console UI and
// the torrent as the one positional argument.
func bbLaunchArgs(install, sep, config string, instancePort, javaMajor int, vmopts []string, extra ...string) []string {
	// ⚠ The heap is the launcher's: the script's 256 MiB on Linux, the
	// vendor's `.vmoptions` on Windows when it names one.
	var args []string
	heap := false
	for _, o := range vmopts {
		heap = heap || strings.HasPrefix(o, "-Xmx")
	}
	if !heap {
		args = append(args, "-Xmx256m")
	}
	args = append(args, vmopts...)
	args = append(args, "-cp", filepath.Join(install, bbJar)+sep+filepath.Join(install, bbCLIJar),
		"-Djava.library.path="+install, "-Dazureus.install.path="+install,
		"-Dazureus.config.path="+config, "-Dazureus.instance.port="+strconv.Itoa(instancePort))
	args = append(args, bbJVMProperties...)
	if javaMajor >= 11 && !bbHasAddOpens(args) {
		args = append(args, "--add-opens", "java.base/java.net=ALL-UNNAMED")
	}
	args = append(args, bbMain)
	return append(args, extra...)
}

func bbHasAddOpens(args []string) bool {
	for i, a := range args {
		if a == "--add-opens" && i+1 < len(args) && args[i+1] == "java.base/java.net=ALL-UNNAMED" {
			return true
		}
		if a == "--add-opens=java.base/java.net=ALL-UNNAMED" {
			return true
		}
	}
	return false
}

// bbVMOptions reads an install4j `.vmoptions` file: one JVM option per line,
// `#` comments, and install4j's own `-include-options` directive, which is not
// a JVM option and is dropped.
func bbVMOptions(text string) []string {
	var out []string
	for _, l := range strings.Split(strings.ReplaceAll(text, "\r", ""), "\n") {
		l = strings.TrimSpace(l)
		if l == "" || strings.HasPrefix(l, "#") || strings.HasPrefix(l, "-include-options") {
			continue
		}
		if strings.HasPrefix(l, "-") {
			out = append(out, l)
		}
	}
	return out
}

// bbJavaEnv is the environment a JVM is started with: the caller's, less the
// three variables a JVM reads options from. ⚠ A host that injects options,
// as this session host's proxy does through JAVA_TOOL_OPTIONS, would change
// the client's network behaviour, so none is passed; `jvm-env.txt` records
// which were dropped.
func bbJavaEnv(env []string, set ...string) (kept []string, dropped []string) {
	drop := map[string]bool{"JAVA_TOOL_OPTIONS": true, "_JAVA_OPTIONS": true, "JDK_JAVA_OPTIONS": true}
	over := map[string]bool{}
	for _, s := range set {
		k, _, _ := strings.Cut(s, "=")
		over[strings.ToUpper(k)] = true
	}
	for _, e := range env {
		k, _, _ := strings.Cut(e, "=")
		switch {
		case drop[strings.ToUpper(k)]:
			dropped = append(dropped, k)
		case over[strings.ToUpper(k)]:
		default:
			kept = append(kept, e)
		}
	}
	return append(kept, set...), dropped
}

// bbCoreState reads one session's console output: "started" once the console
// UI has printed its welcome, which it does from the core's `started` event,
// "failed" once Main has printed *Start fails:*, else "".
func bbCoreState(out string) string {
	switch {
	case strings.Contains(out, "Start fails:"):
		return "failed"
	case strings.Contains(out, "Running BiglyBT "):
		return "started"
	}
	return ""
}

// bbOpened says whether the console UI took the torrent: it prints `Open
// Torrent <path>` when the running instance is handed one.
func bbOpened(out, torrent string) bool {
	for _, l := range strings.Split(strings.ReplaceAll(out, "\r", ""), "\n") {
		if strings.TrimSpace(l) == "Open Torrent "+torrent {
			return true
		}
	}
	return false
}

// -- the source build ---------------------------------------------------------

// bbSourceExcluded is the one source file the build leaves out, as upstream's
// own Maven profile `java-9-plus` deletes it: it implements the Java 8 DNS SPI,
// `sun.net.spi.nameservice`, which no JDK from 9 on has. Its Java 9, 12 and 18
// siblings are compiled.
const bbSourceExcluded = "core/src/com/biglybt/core/util/spi/AENameServiceDescriptor.java"

// bbJavacArgs is the compile, after `javac`: upstream's Maven recipe on JDK 9
// and later, source and target 1.8, ISO-8859-1 sources, against the
// repository's own `core/lib` jars and its Linux SWT jar, which the console UI
// never loads and the compile needs.
func bbJavacArgs(src, classes, sources string) []string {
	cp := strings.Join([]string{filepath.Join(src, "core", "lib", bbCLIJar), filepath.Join(src, "core", "lib", "apple-extensions.jar"),
		filepath.Join(src, "uis", "lib", "swt-linux-64.jar")}, ":")
	return []string{"-J-Xmx1g", "-source", "8", "-target", "8", "-encoding", "ISO-8859-1", "-nowarn", "-Xlint:none",
		"-cp", cp, "-d", classes, "@" + sources}
}

// bbSourceFile sorts one file of `core/src` or `uis/src`: "java" to compile,
// "resource" to copy into the jar, or "" to leave out, as upstream's Maven
// resources exclude `*.java`, `*.class`, `.editorconfig` and the Java 8 DNS
// service file.
func bbSourceFile(rel string) string {
	rel = filepath.ToSlash(rel)
	base := filepath.Base(rel)
	switch {
	case rel == bbSourceExcluded:
		return ""
	case strings.HasSuffix(rel, ".java"):
		return "java"
	case strings.HasSuffix(rel, ".class"), base == ".editorconfig", strings.HasSuffix(base, ".NameServiceDescriptor"):
		return ""
	}
	return "resource"
}

// bbResourceCopies are upstream's antrun copies, made beside the originals:
// Hebrew and Indonesian under their current ISO codes.
var bbResourceCopies = [][2]string{
	{"com/biglybt/internat/MessagesBundle_iw_IL.properties", "com/biglybt/internat/MessagesBundle_he_IL.properties"},
	{"com/biglybt/internat/MessagesBundle_in_ID.properties", "com/biglybt/internat/MessagesBundle_id_ID.properties"},
}

// bbManifest is the source build's manifest: the vendor's jar's, read from
// `GitHub_BiglyBT.jar` 4.1.0.0.
const bbManifest = "Manifest-Version: 1.0\nMain-Class: com.biglybt.ui.Main\nClass-Path: swt.jar commons-cli.jar junit.jar\n"

// -- the tarball --------------------------------------------------------------

// bbTarMember maps one member of the vendor's tarball to its path under the
// install, or refuses it. ⛔ Everything is under `biglybt/`, nothing climbs,
// and only directories and regular files are placed.
func bbTarMember(name string, dir, regular bool) (string, error) {
	rel, ok := strings.CutPrefix(name, "biglybt/")
	if !ok && name != "biglybt" {
		return "", fmt.Errorf("the tarball's member %s is outside biglybt/", name)
	}
	rel = strings.TrimSuffix(rel, "/")
	if !dir && !regular {
		return "", fmt.Errorf("the tarball's member %s is neither a directory nor a file", name)
	}
	if rel == "" {
		if !dir {
			return "", fmt.Errorf("the tarball's member %s is not a directory", name)
		}
		return "", nil
	}
	if strings.HasPrefix(rel, "/") || filepath.IsAbs(rel) {
		return "", fmt.Errorf("the tarball's member %s is absolute", name)
	}
	for _, p := range strings.Split(rel, "/") {
		if p == ".." || p == "." || p == "" {
			return "", fmt.Errorf("the tarball's member %s does not stay under biglybt/", name)
		}
	}
	return rel, nil
}

// -- Windows ------------------------------------------------------------------

// bbInstall4jArgs is the vendor installer's unattended command line, as
// winget's manifest for 4.1 runs it: `-q`, no reboot, and `-dir` naming the
// location. ⚠ An argument vector and not a command line: install4j reads
// `-dir` and the path as two arguments.
func bbInstall4jArgs(dir, log string) ([]string, error) {
	if dir == "" || strings.ContainsAny(dir, "\"\r\n") {
		return nil, fmt.Errorf("the install location [%s] cannot be passed", dir)
	}
	args := []string{"-q", "-dir", dir, "-Dinstall4j.suppressUnattendedReboot=true"}
	if log != "" {
		args = append(args, "-Dinstall4j.log="+log)
	}
	return args, nil
}

// bbJRECfg reads install4j's `inst_jre.cfg` or `pref_jre.cfg`: the JVM home on
// its first line.
func bbJRECfg(text string) string {
	line, _, _ := strings.Cut(strings.ReplaceAll(text, "\r", ""), "\n")
	return strings.TrimSpace(line)
}

// -- the scrub ----------------------------------------------------------------

// bbSecretKey says whether a config key names a credential or a key. Measured
// on 2026-10-21 in a 4.1.0.0 config the client wrote for itself:
// `core.crypto.ecc.1.privatekey` and `.2`, `core.crypto.pw.1.persist_value`,
// `core.crypto.msgsync.*` dictionaries holding `pri`, the RSS plugin's
// `Pairing Session Key` and the simple API's `apikey`. The rest of the names
// are the ones a later build may add.
func bbSecretKey(k string) bool {
	l := strings.ToLower(k)
	for _, s := range []string{"privatekey", "private", "persist_value", "session key", "apikey", "password", "accesscode", "secret", "token"} {
		if strings.Contains(l, s) {
			return true
		}
	}
	return l == "pri"
}

// bbSecretValues collects every value under a secret key, nested anywhere,
// eight bytes or longer, as the bundle scan's value rule counts them.
func bbSecretValues(v any, under bool, out *[][]byte) {
	switch x := v.(type) {
	case map[string]any:
		for k, e := range x {
			bbSecretValues(e, under || bbSecretKey(k), out)
		}
	case []any:
		for _, e := range x {
			bbSecretValues(e, under, out)
		}
	case []byte:
		if under && len(x) >= 8 {
			*out = append(*out, x)
		}
	}
}

// bbScrub empties the client's config directory and keeps its logs.
//
// ⛔ THE WHOLE DIRECTORY GOES, NOT A LIST OF NAMES. A first start writes the
// keys bbSecretKey names into `biglybt.config` and its `.bak`, a Java
// keystore and trust store, `.keystore` and `.certs`, the DHT's node files
// and an anonymous `ID` the version check sends, and the bundle scan names
// none of them. Nothing in it is evidence: the settings are kept as
// `settings.txt`, and `logs/` is moved out to `biglybt-logs/` before the rest
// is removed.
// ⛔ AND THEN EVERY FILE IN THE WORKDIR IS SEARCHED FOR EVERY SECRET VALUE, raw
// and in hex, and removed if it holds one, as `deluge`'s scrub searches for its
// password.
func bbScrub(workdir, config string) ([]string, error) {
	var secrets [][]byte
	for _, name := range []string{"biglybt.config", "biglybt.config.bak"} {
		body, err := os.ReadFile(filepath.Join(config, name))
		if err != nil {
			continue
		}
		if v, err := bdecode(body); err == nil {
			bbSecretValues(v, false, &secrets)
		}
	}
	var needles [][]byte
	for _, s := range secrets {
		needles = append(needles, s, []byte(hex.EncodeToString(s)), []byte(strings.ToUpper(hex.EncodeToString(s))))
	}
	rel := func(p string) string {
		if r, err := filepath.Rel(workdir, p); err == nil {
			return filepath.ToSlash(r)
		}
		return p
	}
	var removed []string
	if fi, err := os.Stat(config); err == nil && fi.IsDir() {
		logs := filepath.Join(config, "logs")
		if fi, err := os.Stat(logs); err == nil && fi.IsDir() {
			kept := filepath.Join(workdir, "biglybt-logs")
			for n := 2; bbExists(kept); n++ {
				kept = filepath.Join(workdir, fmt.Sprintf("biglybt-logs-%d", n))
			}
			if err := os.Rename(logs, kept); err != nil {
				return removed, err
			}
		}
		err := filepath.WalkDir(config, func(p string, d fs.DirEntry, err error) error {
			if err == nil && !d.IsDir() {
				removed = append(removed, rel(p))
			}
			return err
		})
		if err != nil {
			return removed, err
		}
		if err := os.RemoveAll(config); err != nil {
			return removed, err
		}
	}
	if len(needles) > 0 {
		err := filepath.WalkDir(workdir, func(p string, d fs.DirEntry, err error) error {
			if err != nil {
				if os.IsNotExist(err) {
					return nil
				}
				return err
			}
			if !d.Type().IsRegular() {
				return nil
			}
			body, err := os.ReadFile(p)
			if err != nil {
				return err
			}
			for _, n := range needles {
				if bytes.Contains(body, n) {
					if err := os.Remove(p); err != nil {
						return err
					}
					removed = append(removed, rel(p))
					break
				}
			}
			return nil
		})
		if err != nil {
			return removed, err
		}
	}
	sort.Strings(removed)
	return removed, nil
}

// bbStopRecord is the line `stop` appends to `stop.txt`.
func bbStopRecord(at string, pid int, how string) string {
	return fmt.Sprintf("%s pid %d: %s\n", at, pid, how)
}

// -- the selftest -------------------------------------------------------------

// biglybtSelftest is `bit-check biglybt --selftest`.
func biglybtSelftest(args []string) int {
	if len(args) != 1 || args[0] != "--selftest" {
		fmt.Fprintln(os.Stderr, "biglybt: usage: bit-check biglybt --selftest")
		return 2
	}
	cases, failed := 0, 0
	check := func(name string, ok bool) {
		cases++
		if !ok {
			failed++
			fmt.Fprintf(os.Stderr, "biglybt: FAILED %s\n", name)
		}
	}

	// The scheme.
	s := bbDescribeScheme(bbLinuxAsset)
	check("the scheme is v, four components, floor 2", strings.Contains(s, "release_tag_prefix=v\n") &&
		strings.Contains(s, "release_min_components=4\n") && strings.Contains(s, "release_max_components=4\n") &&
		strings.Contains(s, "release_floor_major=2\n") && strings.Contains(s, "release_series=-\n") && strings.Contains(s, "release_repo=BiglySoftware/BiglyBT\n"))
	check("the release digest is the vendor's checksums document", strings.Contains(s, "release_digest_asset=*sha256sum.txt\n") &&
		!strings.Contains(s, "release_digest_unpublished"))
	for _, a := range []string{"GitHub_BiglyBT_unix.tar.gz", "Github_BiglyBT_unix.tar.gz"} {
		check("the Linux asset pattern matches "+a, bbGlob(bbLinuxAsset, a))
	}
	for _, a := range []string{"GitHub_BiglyBT_Installer.sh", "GitHub_BiglyBT.jar", "GitHub_BiglyBT_Windows_Installer.exe"} {
		check("the Linux asset pattern does not match "+a, !bbGlob(bbLinuxAsset, a))
	}
	check("the Windows asset pattern matches the installer with the JRE", bbGlob(bbWindowsAsset, "GitHub_BiglyBT_Windows_Installer.exe"))
	check("and not the NoJRE one", !bbGlob(bbWindowsAsset, "GitHub_BiglyBT_Windows_NoJRE.exe"))
	check("the digest pattern matches both spellings", bbGlob(bbDigestAsset, "GitHub_sha256sum.txt") && bbGlob(bbDigestAsset, "Github_sha256sum.txt"))
	check("the tag v4.1.0.0 names 4.1.0.0", bbTagVersion("v4.1.0.0") == "4.1.0.0")
	check("v1.8.00 and a bare version name nothing", bbTagVersion("v1.8.00") == "" && bbTagVersion("4.1.0.0") == "")
	check("winget's 4.1 is 4.1.0.0", bbSameVersion("4.1", "4.1.0.0") && bbSameVersion("4.1.0.0", "4.1.0.0"))
	check("4.1.0.1 is not 4.1.0.0, nor 4.10 4.1", !bbSameVersion("4.1.0.1", "4.1.0.0") && !bbSameVersion("4.10", "4.1.0.0"))
	check("a version that is not one is never the same", !bbSameVersion("4.1-beta", "4.1.0.0") && !bbSameVersion("", ""))
	for route, want := range map[string]string{"release": "java-tarball", "source": "java-source"} {
		got, ok := bbPackageFormat(false, route)
		check("Linux "+route+" delivers "+want, ok && got == want)
	}
	for route, want := range map[string]string{"release": "install4j-installer", "package": "install4j-installer-nojre"} {
		got, ok := bbPackageFormat(true, route)
		check("Windows "+route+" delivers "+want, ok && got == want)
	}
	_, ok := bbPackageFormat(false, "package")
	check("Linux has no package route", !ok)

	// The settings.
	doc, err := bbConfig("/w/client/downloads", "51413")
	check("the settings encode", err == nil)
	check("the Linux settings are held", err == nil && bbConfigRefusal(doc) == "")
	v, derr := bdecode(doc)
	m, _ := v.(map[string]any)
	check("the document decodes to a dictionary", derr == nil && m != nil)
	for k, want := range map[string]int64{"TCP.Listen.Port": 51413, "UDP.Listen.Port": 51413, "UDP.NonData.Listen.Port": 51413,
		"Plugin.DHT.dht.enabled": 0, "Peer Source Selection Default.PeerExchange": 0,
		"Plugin.azlocaltracker.Plugin.localtracker.enable": 0, "Plugin.UPnP.upnp.enable": 0, "PluginInfo.azutp.enabled": 0} {
		got, _ := m[k].(int64)
		check(fmt.Sprintf("the document holds %s=%d", k, want), m != nil && m[k] != nil && got == want)
	}
	sp, _ := m["Default save path"].([]byte)
	check("the save path is the workdir's", string(sp) == "/w/client/downloads")
	for _, k := range []string{"Tracker Client Send OS and Java Version", "Plugin.azutp.utp.enabled", "network.transport.encrypted.require"} {
		_, set := m[k]
		check("the document leaves "+k+" at the vendor's default", !set)
	}
	wdoc, err := bbConfig(`C:\w\client\downloads`, "51413")
	check("the Windows settings are held", err == nil && bbConfigRefusal(wdoc) == "")
	d0, err := bbConfig("/d", "0")
	m0v, _ := bdecode(d0)
	m0, _ := m0v.(map[string]any)
	_, tcp := m0["TCP.Listen.Port"]
	check("peer port 0 leaves the port to the build", err == nil && m0 != nil && !tcp)
	for _, bad := range []string{"51413 ", "x", "70000", "-1", "051413"} {
		_, err := bbConfig("/d", bad)
		check("the peer port ["+bad+"] is refused", err != nil)
	}
	_, err = bbConfig("", "1")
	check("an empty download directory is refused", err != nil)
	for _, k := range []string{"Plugin.DHT.dht.enabled", "PluginInfo.mldht.enabled", "Plugin.UPnP.natpmp.enable"} {
		mm := map[string]any{}
		for kk, vv := range m {
			switch x := vv.(type) {
			case int64:
				mm[kk] = int(x)
			case []byte:
				mm[kk] = string(x)
			}
		}
		mm[k] = 1
		bad, _ := bencodeDict(mm)
		check("a document with "+k+" on is refused", strings.Contains(bbConfigRefusal(bad), k))
		delete(mm, k)
		missing, _ := bencodeDict(mm)
		check("a document with no "+k+" is refused", strings.Contains(bbConfigRefusal(missing), k))
	}
	check("a document that is not bencoded is refused", bbConfigRefusal([]byte("d3:abc")) != "")
	text := bbSettingsText(doc)
	check("the settings are kept as text", strings.Contains(text, "Plugin.DHT.dht.enabled=0\n") && strings.Contains(text, "Default save path=/w/client/downloads\n"))
	check("the instance discovery is off by property", len(bbJVMProperties) == 1 && bbJVMProperties[0] == "-Daz.instance.manager.enable=0")
	check("eleven bundled plugins, uTP among them", len(bbBundledPlugins) == 11 && strings.Contains(strings.Join(bbBundledPlugins, ","), "azutp"))

	// Bencoding.
	for in, ok := range map[string]bool{"de": true, "d1:ai1ee": true, "d1:a2:bce": true, "li1ei2ee": true, "d1:ai1e": false, "i1": false,
		"2:a": false, "d1:ai1eex": false, "di1ei1ee": false, "x": false} {
		_, err := bdecode([]byte(in))
		check(fmt.Sprintf("bencoded [%s] decodes: %v", in, ok), (err == nil) == ok)
	}
	_, err = bencodeDict(map[string]any{"a": 1.5})
	check("a setting that is neither an int nor a string is refused", err != nil)

	// The probe class.
	cls := bbVersionClass()
	check("the probe is a class file, version 52", len(cls) > 10 && bytes.Equal(cls[:4], []byte{0xca, 0xfe, 0xba, 0xbe}) &&
		binary.BigEndian.Uint16(cls[6:8]) == 52)
	for _, needle := range []string{bbProbeClass, "com.biglybt.core.util.Constants", "BIGLYBT_VERSION", "VERSION_ID",
		"java.vendor", "java.runtime.version", "getField", "forName", "([B)V"} {
		check("the probe names "+needle, bytes.Contains(cls, []byte(needle)))
	}
	check("the probe does not name the version it is asking for", !bytes.Contains(cls, []byte("4.1.0.0")))
	check("the probe is the same bytes every time", bytes.Equal(cls, bbVersionClass()))
	check("the probe ends in its three empty tables", bytes.HasSuffix(cls, []byte{0xb1, 0, 0, 0, 0, 0, 0}))

	// The build's answer.
	answer := "biglybt.version=4.1.0.0\nbiglybt.peer_id_prefix=-BI4100-\njava.version=21.0.12.1\njava.vendor=Ubuntu\n" +
		"java.vm.name=OpenJDK 64-Bit Server VM\njava.runtime.version=21.0.12.1+1-1-24.04.4-Ubuntu\nos.name=Linux\nos.arch=amd64\n"
	a, err := bbParseAnswer(answer)
	check("the answer reads 4.1.0.0, -BI4100- on Java 21.0.12.1", err == nil && a.version == "4.1.0.0" && a.peerPrefix == "-BI4100-" &&
		a.java == "21.0.12.1" && a.fields["java.vendor"] == "Ubuntu")
	_, err = bbParseAnswer(strings.ReplaceAll(answer, "\n", "\r\n"))
	check("a Windows answer reads the same", err == nil)
	for name, bad := range map[string]string{
		"no version":           strings.Replace(answer, "biglybt.version=4.1.0.0", "biglybt.version=null", 1),
		"a three-part version": strings.Replace(answer, "biglybt.version=4.1.0.0", "biglybt.version=4.1.0", 1),
		"another peer prefix":  strings.Replace(answer, "-BI4100-", "-BI4000-", 1),
		"a foreign prefix":     strings.Replace(answer, "-BI4100-", "-AZ4100-", 1),
		"no Java":              strings.Replace(answer, "java.version=21.0.12.1", "java.version=null", 1),
		"a doubled key":        answer + "biglybt.version=4.0.0.0\n",
		"nothing":              "",
	} {
		_, err := bbParseAnswer(bad)
		check("an answer with "+name+" is refused", err != nil)
	}
	for in, want := range map[string]int{"1.8.0_462": 8, "17.0.16": 17, "21.0.12.1": 21, "25": 25, "11-ea": 11} {
		check(fmt.Sprintf("Java %s is %d", in, want), bbJavaMajor(in) == want)
	}

	// The launch.
	la := bbLaunchArgs("/p/opt/biglybt", ":", "/w/config", 40123, 21, nil, "--ui=console", "/w/t.torrent")
	j := strings.Join(la, " ")
	check("the launch is the vendor's: -Xmx256m, the jar and commons-cli, the install path",
		la[0] == "-Xmx256m" && strings.Contains(j, "-cp /p/opt/biglybt/BiglyBT.jar:/p/opt/biglybt/commons-cli.jar") &&
			strings.Contains(j, "-Dazureus.install.path=/p/opt/biglybt") && strings.Contains(j, "-Djava.library.path=/p/opt/biglybt"))
	check("the launch opens java.net on Java 11 and later, once", strings.Count(j, "--add-opens java.base/java.net=ALL-UNNAMED") == 1)
	check("the launch carries the capture's config, instance port and instance switch",
		strings.Contains(j, "-Dazureus.config.path=/w/config") && strings.Contains(j, "-Dazureus.instance.port=40123") &&
			strings.Contains(j, "-Daz.instance.manager.enable=0"))
	check("the launch ends in the console UI and the torrent", strings.HasSuffix(j, "com.biglybt.ui.Main --ui=console /w/t.torrent"))
	l8 := strings.Join(bbLaunchArgs("/p", ":", "/c", 1, 8, nil), " ")
	check("Java 8 gets no --add-opens", !strings.Contains(l8, "--add-opens"))
	vm := bbVMOptions("# Enter one VM parameter per line\r\n-Xmx512m\r\n--add-opens=java.base/java.net=ALL-UNNAMED\r\n-include-options user.vmoptions\r\n\r\n")
	check("the vendor's vmoptions are read, comments and install4j's directive dropped", strings.Join(vm, " ") == "-Xmx512m --add-opens=java.base/java.net=ALL-UNNAMED")
	lw := strings.Join(bbLaunchArgs(`C:\P\BiglyBT`, ";", `C:\w\config`, 2, 21, vm), " ")
	check("on Windows the vendor's heap is the one, and its --add-opens is not doubled",
		strings.Count(lw, "-Xmx") == 1 && strings.Contains(lw, "-Xmx512m") && strings.Count(lw, "add-opens") == 1)
	env, dropped := bbJavaEnv([]string{"PATH=/bin", "JAVA_TOOL_OPTIONS=-Dhttps.proxyHost=x", "HOME=/root", "_JAVA_OPTIONS=-Dx"}, "HOME=/w/home")
	check("the JVM's option variables are dropped and recorded", strings.Join(env, " ") == "PATH=/bin HOME=/w/home" &&
		strings.Join(dropped, ",") == "JAVA_TOOL_OPTIONS,_JAVA_OPTIONS")

	// The core and the handover.
	check("a welcome is a started core", bbCoreState("BiglyBT started at x\nRunning BiglyBT 4.1.0.0...\n") == "started")
	check("Start fails is a failed core, even after a welcome", bbCoreState("Running BiglyBT 4.1.0.0...\nStart fails:\n") == "failed")
	check("nothing yet is neither", bbCoreState("BiglyBT started at 22-Oct-2026\n") == "")
	check("the console's Open Torrent line is the handover taken", bbOpened("> x\r\nOpen Torrent /w/run.torrent\r\n", "/w/run.torrent"))
	check("another torrent is not", !bbOpened("Open Torrent /w/other.torrent\n", "/w/run.torrent"))

	// The source build.
	for rel, want := range map[string]string{
		"core/src/com/biglybt/core/util/Constants.java": "java",
		bbSourceExcluded: "",
		"core/src/com/biglybt/core/util/spi/AENameServiceJava18.java":              "java",
		"core/src/META-INF/services/sun.net.spi.nameservice.NameServiceDescriptor": "",
		"uis/src/com/biglybt/ui/icons/a.png":                                       "resource",
		"core/src/com/biglybt/internat/MessagesBundle.properties":                  "resource",
		"core/src/.editorconfig":                                                   "",
		"core/src/x/Y.class":                                                       "",
	} {
		check(fmt.Sprintf("the source file %s is %q", rel, want), bbSourceFile(rel) == want)
	}
	ja := strings.Join(bbJavacArgs("/s", "/s/classes", "/s/sources.txt"), " ")
	check("javac compiles source and target 8, ISO-8859-1, against the tag's own jars", strings.Contains(ja, "-source 8 -target 8") &&
		strings.Contains(ja, "-encoding ISO-8859-1") && strings.Contains(ja, "/s/core/lib/commons-cli.jar:/s/core/lib/apple-extensions.jar:/s/uis/lib/swt-linux-64.jar") &&
		strings.HasSuffix(ja, "-d /s/classes @/s/sources.txt"))
	check("upstream's two locale copies are made", len(bbResourceCopies) == 2 && strings.HasSuffix(bbResourceCopies[0][1], "he_IL.properties"))
	check("the manifest is the vendor jar's", strings.Contains(bbManifest, "Main-Class: com.biglybt.ui.Main\n"))

	// The tarball.
	for in, want := range map[string]string{"biglybt/": "", "biglybt/BiglyBT.jar": "BiglyBT.jar", "biglybt/plugins/azutp/azutp_0.7.8.jar": "plugins/azutp/azutp_0.7.8.jar"} {
		got, err := bbTarMember(in, strings.HasSuffix(in, "/"), !strings.HasSuffix(in, "/"))
		check("the tarball's "+in+" is placed at ["+want+"]", err == nil && got == want)
	}
	for _, bad := range []string{"etc/passwd", "biglybt/../x", "biglybt//x", "/biglybt/x", "biglybt/./x"} {
		_, err := bbTarMember(bad, false, true)
		check("the tarball's member "+bad+" is refused", err != nil)
	}
	_, err = bbTarMember("biglybt/link", false, false)
	check("a member that is neither a file nor a directory is refused", err != nil)

	// Windows.
	ia, err := bbInstall4jArgs(`C:\Program Files\BiglyBT`, `C:\w\install4j.log`)
	check("the installer runs unattended into the location", err == nil &&
		strings.Join(ia, "|") == `-q|-dir|C:\Program Files\BiglyBT|-Dinstall4j.suppressUnattendedReboot=true|-Dinstall4j.log=C:\w\install4j.log`)
	_, err = bbInstall4jArgs(`C:\a"b`, "")
	check("a location holding a quote is refused", err != nil)
	check("install4j's JRE file names its home", bbJRECfg("C:\\Program Files\\Eclipse Adoptium\\jre-25.0.1.8-hotspot\r\n") == `C:\Program Files\Eclipse Adoptium\jre-25.0.1.8-hotspot`)

	// The scrub, over a scratch workdir.
	work, err := os.MkdirTemp("", "biglybt-selftest-")
	if err != nil {
		fmt.Fprintln(os.Stderr, "biglybt: cannot make a scratch directory")
		return 2
	}
	defer os.RemoveAll(work)
	mk := func(rel string, body []byte) {
		p := filepath.Join(work, filepath.FromSlash(rel))
		os.MkdirAll(filepath.Dir(p), 0o755)
		os.WriteFile(p, body, 0o644)
	}
	key := []byte("\x00\x00\x00\x02private-key-bytes-of-the-client")
	api := "zyovvxgesqnorcbzjcqztyfarwljblrah"
	cfg, _ := bencodeDict(map[string]any{"core.crypto.ecc.1.privatekey": string(key), "Plugin.azintsimpleapi.apikey": api,
		"Plugin.DHT.dht.enabled": 0, "ID": "XRv9Djn2XQ35IVPsElVh"})
	nested := []byte("d26:core.crypto.msgsync.Publicd3:pri24:nested-private-key-bytes3:pub3:abcee")
	mk("config/biglybt.config", cfg)
	mk("config/biglybt.config.bak", nested)
	mk("config/.keystore", []byte("keystore"))
	mk("config/dht/contacts.dat", []byte("dht"))
	mk("config/logs/debug_1.log", []byte("[00:39] VersionCheckClient - UnknownHostException\n"))
	mk("client.out", []byte("Running BiglyBT 4.1.0.0...\n"))
	mk("stray.txt", []byte("the api key "+api+" leaked\n"))
	mk("hexed.txt", []byte(hex.EncodeToString(key)))
	mk("nested.txt", []byte("nested-private-key-bytes"))
	mk("downloads/payload.bin", []byte{0, 1})
	gone, err := bbScrub(work, filepath.Join(work, "config"))
	check("the scrub removes the whole config and every file holding a secret", err == nil && strings.Join(gone, ",") ==
		"config/.keystore,config/biglybt.config,config/biglybt.config.bak,config/dht/contacts.dat,hexed.txt,nested.txt,stray.txt")
	check("the scrub keeps the logs, moved out, and the rest", isRegular(filepath.Join(work, "biglybt-logs", "debug_1.log")) &&
		isRegular(filepath.Join(work, "client.out")) && isRegular(filepath.Join(work, "downloads", "payload.bin")) && !bbExists(filepath.Join(work, "config")))
	gone, err = bbScrub(work, filepath.Join(work, "config"))
	check("a second scrub finds nothing", err == nil && len(gone) == 0)
	mk("config/logs/debug_1.log", []byte("second session\n"))
	_, err = bbScrub(work, filepath.Join(work, "config"))
	check("a second session's logs are kept beside the first's", err == nil && isRegular(filepath.Join(work, "biglybt-logs-2", "debug_1.log")))
	check("a short value is not a secret", func() bool {
		var o [][]byte
		bbSecretValues(map[string]any{"apikey": []byte("short")}, false, &o)
		return len(o) == 0
	}())
	check("stop's record says how", bbStopRecord("2026-10-21T00:00:00Z", 7, "sigterm") == "2026-10-21T00:00:00Z pid 7: sigterm\n")

	fmt.Printf(`{"schema":"biglybt-selftest/1","cases":%d,"failed":%d}`+"\n", cases, failed)
	if failed > 0 {
		return 1
	}
	return 0
}

// bbExists is `[ -e PATH ]`, here because the kit's is Linux-only.
func bbExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// bbGlob is `release_asset`'s two constructs, `*` and literal text, as the
// selftest asks them of the patterns above. ⚠ The resolver's own matcher is
// the authority; this only proves the patterns say what this file means.
func bbGlob(pattern, name string) bool {
	parts := strings.Split(pattern, "*")
	if !strings.HasPrefix(name, parts[0]) {
		return false
	}
	rest := name[len(parts[0]):]
	for i := 1; i < len(parts); i++ {
		if i == len(parts)-1 {
			return strings.HasSuffix(rest, parts[i]) && len(rest) >= len(parts[i])
		}
		k := strings.Index(rest, parts[i])
		if k < 0 {
			return false
		}
		rest = rest[k+len(parts[i]):]
	}
	return rest == ""
}

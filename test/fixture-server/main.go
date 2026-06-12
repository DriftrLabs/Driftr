// Command fixture-server serves a minimal fake of nodejs.org/dist for
// hermetic integration tests. Point driftr at it with
// DRIFTR_NODE_MIRROR=http://127.0.0.1:<port>.
//
// It serves a single fake Node.js version for every os/arch combination:
//
//	/index.json                                  release index
//	/v<ver>/SHASUMS256.txt                       checksums for all platforms
//	/v<ver>/node-v<ver>-<os>-<arch>.tar.gz       fake tarball
//
// The tarballs contain shell scripts for bin/node, bin/npm, and bin/npx that
// print plausible version strings, so shim-execution tests work without a
// real Node.js download.
package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"flag"
	"fmt"
	"log"
	"net/http"
	"sort"
)

const fixtureVersion = "22.99.0"

var platforms = []struct{ os, arch string }{
	{"linux", "x64"},
	{"linux", "arm64"},
	{"darwin", "x64"},
	{"darwin", "arm64"},
}

// buildTarball creates a fake Node.js release archive for one platform.
func buildTarball(osName, arch string) ([]byte, error) {
	prefix := fmt.Sprintf("node-v%s-%s-%s/", fixtureVersion, osName, arch)
	scripts := map[string]string{
		"bin/node": fmt.Sprintf("#!/bin/sh\necho \"v%s\"\n", fixtureVersion),
		"bin/npm":  "#!/bin/sh\necho \"10.99.0\"\n",
		"bin/npx":  "#!/bin/sh\necho \"10.99.0\"\n",
	}

	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)

	names := make([]string, 0, len(scripts))
	for name := range scripts {
		names = append(names, name)
	}
	sort.Strings(names) // deterministic archive bytes

	for _, name := range names {
		content := scripts[name]
		hdr := &tar.Header{
			Name:     prefix + name,
			Mode:     0o755,
			Size:     int64(len(content)),
			Typeflag: tar.TypeReg,
		}
		if err := tw.WriteHeader(hdr); err != nil {
			return nil, err
		}
		if _, err := tw.Write([]byte(content)); err != nil {
			return nil, err
		}
	}
	if err := tw.Close(); err != nil {
		return nil, err
	}
	if err := gz.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func main() {
	addr := flag.String("addr", "127.0.0.1:9123", "listen address")
	flag.Parse()

	tarballs := map[string][]byte{} // filename -> archive bytes
	var shasums bytes.Buffer
	for _, p := range platforms {
		data, err := buildTarball(p.os, p.arch)
		if err != nil {
			log.Fatalf("build tarball %s-%s: %v", p.os, p.arch, err)
		}
		filename := fmt.Sprintf("node-v%s-%s-%s.tar.gz", fixtureVersion, p.os, p.arch)
		tarballs[filename] = data
		fmt.Fprintf(&shasums, "%x  %s\n", sha256.Sum256(data), filename)
	}

	index := fmt.Sprintf(`[{"version":"v%s","lts":false}]`, fixtureVersion)

	mux := http.NewServeMux()
	mux.HandleFunc("/index.json", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, index)
	})
	mux.HandleFunc(fmt.Sprintf("/v%s/SHASUMS256.txt", fixtureVersion), func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(shasums.Bytes())
	})
	mux.HandleFunc(fmt.Sprintf("/v%s/", fixtureVersion), func(w http.ResponseWriter, r *http.Request) {
		name := r.URL.Path[len(fmt.Sprintf("/v%s/", fixtureVersion)):]
		data, ok := tarballs[name]
		if !ok {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/gzip")
		_, _ = w.Write(data)
	})

	log.Printf("fixture server listening on %s (node v%s)", *addr, fixtureVersion)
	log.Fatal(http.ListenAndServe(*addr, mux))
}

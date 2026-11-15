package core

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	goruntime "runtime"
	"strings"
	"unicode"

	"github.com/nsgo-dev/nsgo/internal/driver"
)

// FileFromURL returns the file a URL from OnFileDrop, ReadClipboardFiles or a PasteEvent names, for File.Open to read. A file: URL names a local file, which has its path, size and modification time; the web's blob: URL, whose fragment is the file's name, names a file the page holds, which has no path, and has its size, type and time when the page made the URL from a drop. A folder, a file that is not there, a file: URL with a remote host outside Windows, a blob: URL off the web and every other scheme return an error.
func FileFromURL(raw string) (File, error) {
	if raw == "" {
		return File{}, errors.New("core: empty file URL")
	}
	if strings.IndexFunc(raw, func(r rune) bool { return unicode.IsSpace(r) || unicode.IsControl(r) }) >= 0 {
		return File{}, errors.New("core: file URL has spaces or control characters")
	}
	u, err := url.Parse(raw)
	if err != nil {
		return File{}, fmt.Errorf("core: invalid file URL: %w", err)
	}
	switch u.Scheme {
	case "file":
		return localFileFromURL(u)
	case "blob":
		return blobFileFromURL(u)
	case "":
		return File{}, errors.New("core: relative URLs are not files")
	default:
		return File{}, fmt.Errorf("core: %s URLs are not files", u.Scheme)
	}
}

func localFileFromURL(u *url.URL) (File, error) {
	if u.Opaque != "" || u.User != nil || u.RawQuery != "" || u.ForceQuery || u.Fragment != "" {
		return File{}, errors.New("core: file URL has unsupported components")
	}
	path, err := filePath(u, goruntime.GOOS == "windows")
	if err != nil {
		return File{}, err
	}
	info, err := os.Stat(path)
	if err != nil {
		return File{}, fmt.Errorf("core: %w", err)
	}
	if info.IsDir() {
		return File{}, fmt.Errorf("core: %q is a folder", path)
	}
	name := filepath.Base(path)
	return File{Name: name, MIME: driver.MIMEForFile(name), Size: info.Size(), Modified: info.ModTime(), Path: path}, nil
}

// filePath is the path a file: URL names on Windows, where /C:/ names a drive, with no host or localhost, and a host a UNC share, or elsewhere, where only localhost may be its host.
func filePath(u *url.URL, windows bool) (string, error) {
	if !strings.HasPrefix(u.Path, "/") {
		return "", errors.New("core: file URL has no absolute path")
	}
	local := u.Host == "" || strings.EqualFold(u.Host, "localhost")
	if !windows {
		if !local {
			return "", fmt.Errorf("core: file URL names a file on %s", u.Host)
		}
		return u.Path, nil
	}
	p := u.Path
	drive := len(p) >= 3 && p[2] == ':' && 'a' <= p[1]|0x20 && p[1]|0x20 <= 'z'
	switch {
	case drive && local:
		p = p[1:]
	case u.Host != "":
		// The share \\localhost\C$ is written file://localhost/C$.
		p = "//" + u.Host + p
	}
	return strings.ReplaceAll(p, "/", `\`), nil
}

func blobFileFromURL(u *url.URL) (File, error) {
	if u.Opaque == "" || u.User != nil || u.RawQuery != "" || u.ForceQuery {
		return File{}, errors.New("core: invalid blob URL")
	}
	name := u.Fragment
	if name == "" || name == "." || name == ".." || strings.ContainsAny(name, `/\`) {
		return File{}, errors.New("core: blob URL has no file name")
	}
	if driver.BlobFile == nil {
		return File{}, errors.New("core: blob URLs are files only on the web")
	}
	return driver.BlobFile("blob:"+u.Opaque, name), nil
}

package static

import (
	"archive/tar"
	"bytes"
	"debug/elf"
	"io"
	"os/exec"
	"strings"
	"testing"
)

// T2.1-6
func TestServiceImagesStatic(t *testing.T) {
	if _, err := exec.LookPath("docker"); err != nil {
		t.Skip("docker not installed")
	}
	for _, svc := range []string{"inventory"} {
		t.Run(svc, func(t *testing.T) {
			tag := "grafana-lab-static-test/" + svc
			run(t, "docker", "build", "-q", "--target", svc, "-t", tag, "../../services")
			id := strings.TrimSpace(run(t, "docker", "create", tag))
			defer exec.Command("docker", "rm", id).Run()

			files := map[string][]byte{}
			tr := tar.NewReader(strings.NewReader(run(t, "docker", "export", id)))
			for {
				h, err := tr.Next()
				if err == io.EOF {
					break
				}
				if err != nil {
					t.Fatal(err)
				}
				name := strings.TrimPrefix(h.Name, "/")
				if h.Typeflag == tar.TypeReg || h.Typeflag == tar.TypeSymlink {
					b, _ := io.ReadAll(tr)
					files[name] = b
				}
			}
			for name := range files {
				if base := name[strings.LastIndex(name, "/")+1:]; base == "sh" || base == "bash" || base == "busybox" {
					t.Errorf("image contains a shell: %s", name)
				}
			}
			bin, ok := files[svc]
			if !ok {
				t.Fatalf("image has no /%s; files: %v", svc, keys(files))
			}
			f, err := elf.NewFile(bytes.NewReader(bin))
			if err != nil {
				t.Fatal(err)
			}
			for _, p := range f.Progs {
				if p.Type == elf.PT_INTERP {
					t.Errorf("/%s is dynamically linked", svc)
				}
			}
			if libs, _ := f.ImportedLibraries(); len(libs) > 0 {
				t.Errorf("/%s imports %v", svc, libs)
			}
		})
	}
}

func run(t *testing.T, name string, args ...string) string {
	t.Helper()
	var stderr bytes.Buffer
	cmd := exec.Command(name, args...)
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("%s %v: %v\n%s", name, args, err, stderr.String())
	}
	return string(out)
}

func keys(m map[string][]byte) []string {
	var ks []string
	for k := range m {
		ks = append(ks, k)
	}
	return ks
}

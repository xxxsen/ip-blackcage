package ipset

import (
	"bytes"
	"context"
	"encoding/xml"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"

	"github.com/google/uuid"
)

var (
	defaultVersionRegexp = regexp.MustCompile(`ipset\s+(v.*),\s+protocol\s+version:\s+(.*)`)
)

type errPack struct {
	err    error
	stdout []byte
	stderr []byte
}

type IPSet struct {
	path string
}

func New() (*IPSet, error) {
	path, err := exec.LookPath("ipset")
	if err != nil {
		return nil, fmt.Errorf("lookup ipset command failed, err:%w", err)
	}
	return &IPSet{path: path}, nil
}

func MustNew() *IPSet {
	set, err := New()
	if err != nil {
		panic(err)
	}
	return set
}

func (s *IPSet) runCmd(ctx context.Context, c *config, args ...string) *errPack {
	if len(c.params) > 0 {
		newArgs := make([]string, 0, len(args)+len(c.params))
		newArgs = append(newArgs, args...)
		newArgs = append(newArgs, c.params...)
		args = newArgs
	}
	cmd := exec.CommandContext(ctx, s.path, args...)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	info := &errPack{
		err:    err,
		stdout: stdout.Bytes(),
		stderr: stderr.Bytes(),
	}
	return info
}

func (s *IPSet) runCmdNoData(ctx context.Context, c *config, args ...string) error {
	pack := s.runCmd(ctx, c, args...)
	return pack.err
}

func (s *IPSet) Create(ctx context.Context, set string, typ SetType, opts ...CmdOption) error {
	return s.runCmdNoData(ctx, applyOpts(opts...), "create", set, string(typ))
}

func (s *IPSet) Destroy(ctx context.Context, set string, opts ...CmdOption) error {
	return s.runCmdNoData(ctx, applyOpts(opts...), "destroy", set)
}

func (s *IPSet) Add(ctx context.Context, set string, data string, opts ...CmdOption) error {
	return s.runCmdNoData(ctx, applyOpts(opts...), "add", set, data)
}

func (s *IPSet) Del(ctx context.Context, set string, data string, opts ...CmdOption) error {
	return s.runCmdNoData(ctx, applyOpts(opts...), "del", set, data)
}

func (s *IPSet) ListRaw(ctx context.Context, set string, opts ...CmdOption) ([]byte, error) {
	pack := s.runCmd(ctx, applyOpts(opts...), "list", set)
	if pack.err != nil {
		return nil, pack.err
	}
	return pack.stdout, nil
}

func (s *IPSet) List(ctx context.Context, set string) (*Header, []string, error) {
	raw, err := s.ListRaw(ctx, set, WithOutput("xml"))
	if err != nil {
		return nil, nil, err
	}
	var ipsets Ipsets
	if err := xml.Unmarshal(raw, &ipsets); err != nil {
		return nil, nil, err
	}
	if len(ipsets.Ipset) == 0 {
		return nil, nil, fmt.Errorf("invalid ipset output struct, no ipset data")
	}
	ipset := ipsets.Ipset[0]
	ips := make([]string, 0, len(ipset.Members.Member))
	for _, item := range ipset.Members.Member {
		ips = append(ips, item.Elem)
	}
	return &ipset.Header, ips, nil

}

func (s *IPSet) Restore(ctx context.Context, set string, ips []string, opts ...CmdOption) error {
	buf := bytes.Buffer{}
	for _, ip := range ips {
		buf.WriteString(fmt.Sprintf("add %s %s -exist\n", set, ip))
	}
	tmpDir := os.TempDir()
	tmpName := "blackips-" + uuid.NewString()
	tmpPath := filepath.Join(tmpDir, tmpName)
	defer os.RemoveAll(tmpPath)
	if err := os.WriteFile(tmpPath, buf.Bytes(), 0644); err != nil {
		return fmt.Errorf("write black list to tmp file failed, err:%w", err)
	}
	if err := s.runCmdNoData(ctx, applyOpts(opts...), "restore", "-f", tmpPath); err != nil {
		return err
	}
	return nil

}

func (s *IPSet) Test(ctx context.Context, set string, data string, opts ...CmdOption) (bool, error) {
	pack := s.runCmd(ctx, applyOpts(opts...), "test", set, data)
	if pack.err == nil {
		return bytes.Contains(pack.stderr, []byte("is in set")), nil
	}
	if bytes.Contains(pack.stderr, []byte("is NOT in")) {
		return false, nil
	}
	return false, fmt.Errorf("call cmd failed, err:%w, debug:%s", pack.err, string(pack.stderr))

}

func (s *IPSet) Rename(ctx context.Context, olds, news string, opts ...CmdOption) error {
	return s.runCmdNoData(ctx, applyOpts(opts...), "rename", olds, news)
}

func (s *IPSet) Swap(ctx context.Context, olds, news string, opts ...CmdOption) error {
	return s.runCmdNoData(ctx, applyOpts(opts...), "swap", olds, news)
}

func (s *IPSet) Flush(ctx context.Context, set string, opts ...CmdOption) error {
	return s.runCmdNoData(ctx, applyOpts(opts...), "flush", set)
}

func (s *IPSet) Version(ctx context.Context, opts ...CmdOption) (string, string, error) {
	pack := s.runCmd(ctx, applyOpts(opts...), "version")
	if pack.err != nil {
		return "", "", pack.err
	}
	out := defaultVersionRegexp.FindStringSubmatch(string(pack.stdout))
	if len(out) != 3 {
		return "", "", fmt.Errorf("invalid version format:%s", string(pack.stdout))
	}
	return out[1], out[2], nil
}

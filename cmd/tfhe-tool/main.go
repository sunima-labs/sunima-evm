// tfhe-tool is a dev-only helper that exposes raw tfhebridge
// primitives over a tiny CLI. Used to drive the Stage 5.1 single-node
// devnet end-to-end check: generate keys, encrypt plaintexts off-chain
// for tx submission, and decrypt the homomorphic-add result back.
//
// All output goes to plain files. Nothing here is reachable from the
// chain or from a published binary — this stays out of the consensus
// path on purpose.
//
// Usage:
//
//	tfhe-tool keygen --out-dir <dir>
//	tfhe-tool encrypt --client-key <path> --value <u64> --out <path>
//	tfhe-tool decrypt --client-key <path> --in <path>
//	tfhe-tool ctid    --in <path>
//	tfhe-tool add     --server-key <path> --a <path> --b <path> --out <path>
package main

import (
	"crypto/sha256"
	"encoding/hex"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/sunima-labs/sunima-evm/internal/tfhebridge"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	switch os.Args[1] {
	case "keygen":
		mustRun(cmdKeygen(os.Args[2:]))
	case "encrypt":
		mustRun(cmdEncrypt(os.Args[2:]))
	case "decrypt":
		mustRun(cmdDecrypt(os.Args[2:]))
	case "ctid":
		mustRun(cmdCtID(os.Args[2:]))
	case "add":
		mustRun(cmdAdd(os.Args[2:]))
	default:
		usage()
		os.Exit(2)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: tfhe-tool {keygen|encrypt|decrypt|ctid} [flags]")
}

func mustRun(err error) {
	if err == nil {
		return
	}
	fmt.Fprintf(os.Stderr, "FAIL: %v\n", err)
	os.Exit(1)
}

func cmdKeygen(args []string) error {
	fs := flag.NewFlagSet("keygen", flag.ExitOnError)
	outDir := fs.String("out-dir", "", "directory to write client_key.bin and server_key.bin into")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *outDir == "" {
		return fmt.Errorf("--out-dir is required")
	}
	if err := os.MkdirAll(*outDir, 0o755); err != nil {
		return fmt.Errorf("mkdir: %w", err)
	}
	ck, sk, err := tfhebridge.Keygen()
	if err != nil {
		return fmt.Errorf("keygen: %w", err)
	}
	ckPath := filepath.Join(*outDir, "client_key.bin")
	skPath := filepath.Join(*outDir, "server_key.bin")
	if err := os.WriteFile(ckPath, ck, 0o600); err != nil {
		return fmt.Errorf("write client key: %w", err)
	}
	if err := os.WriteFile(skPath, sk, 0o644); err != nil {
		return fmt.Errorf("write server key: %w", err)
	}
	fmt.Printf("client_key %d B  -> %s\n", len(ck), ckPath)
	fmt.Printf("server_key %d B  -> %s\n", len(sk), skPath)
	return nil
}

func cmdEncrypt(args []string) error {
	fs := flag.NewFlagSet("encrypt", flag.ExitOnError)
	ckPath := fs.String("client-key", "", "path to client_key.bin")
	value := fs.Uint64("value", 0, "u64 plaintext")
	outPath := fs.String("out", "", "path to write ciphertext bytes")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *ckPath == "" || *outPath == "" {
		return fmt.Errorf("--client-key and --out are required")
	}
	ck, err := os.ReadFile(*ckPath)
	if err != nil {
		return fmt.Errorf("read client key: %w", err)
	}
	ct, err := tfhebridge.EncryptU64(ck, *value)
	if err != nil {
		return fmt.Errorf("encrypt: %w", err)
	}
	if err := os.WriteFile(*outPath, ct, 0o644); err != nil {
		return fmt.Errorf("write ct: %w", err)
	}
	id := sha256.Sum256(ct)
	fmt.Printf("ct %d B -> %s\n", len(ct), *outPath)
	fmt.Printf("ctid sha256=%s\n", hex.EncodeToString(id[:]))
	return nil
}

func cmdDecrypt(args []string) error {
	fs := flag.NewFlagSet("decrypt", flag.ExitOnError)
	ckPath := fs.String("client-key", "", "path to client_key.bin")
	inPath := fs.String("in", "", "path to ciphertext bytes")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *ckPath == "" || *inPath == "" {
		return fmt.Errorf("--client-key and --in are required")
	}
	ck, err := os.ReadFile(*ckPath)
	if err != nil {
		return fmt.Errorf("read client key: %w", err)
	}
	ct, err := os.ReadFile(*inPath)
	if err != nil {
		return fmt.Errorf("read ct: %w", err)
	}
	got, err := tfhebridge.DecryptU64(ck, ct)
	if err != nil {
		return fmt.Errorf("decrypt: %w", err)
	}
	fmt.Println(strconv.FormatUint(got, 10))
	return nil
}

func cmdAdd(args []string) error {
	fs := flag.NewFlagSet("add", flag.ExitOnError)
	skPath := fs.String("server-key", "", "path to server_key.bin")
	aPath := fs.String("a", "", "path to ciphertext A")
	bPath := fs.String("b", "", "path to ciphertext B")
	outPath := fs.String("out", "", "path to write result ciphertext")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *skPath == "" || *aPath == "" || *bPath == "" || *outPath == "" {
		return fmt.Errorf("--server-key, --a, --b, --out are required")
	}
	sk, err := os.ReadFile(*skPath)
	if err != nil {
		return fmt.Errorf("read server key: %w", err)
	}
	a, err := os.ReadFile(*aPath)
	if err != nil {
		return fmt.Errorf("read a: %w", err)
	}
	b, err := os.ReadFile(*bPath)
	if err != nil {
		return fmt.Errorf("read b: %w", err)
	}
	out, err := tfhebridge.AddU64(sk, a, b)
	if err != nil {
		return fmt.Errorf("add: %w", err)
	}
	if err := os.WriteFile(*outPath, out, 0o644); err != nil {
		return fmt.Errorf("write out: %w", err)
	}
	id := sha256.Sum256(out)
	fmt.Printf("ct %d B -> %s\n", len(out), *outPath)
	fmt.Printf("ctid sha256=%s\n", hex.EncodeToString(id[:]))
	return nil
}

func cmdCtID(args []string) error {
	fs := flag.NewFlagSet("ctid", flag.ExitOnError)
	inPath := fs.String("in", "", "path to ciphertext bytes")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *inPath == "" {
		return fmt.Errorf("--in is required")
	}
	ct, err := os.ReadFile(*inPath)
	if err != nil {
		return fmt.Errorf("read ct: %w", err)
	}
	id := sha256.Sum256(ct)
	fmt.Println(hex.EncodeToString(id[:]))
	return nil
}

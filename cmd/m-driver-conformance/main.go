// Command m-driver-conformance is the executable contract gate (driver-contract.md
// §9): it drives a driver binary over the subprocess + JSON-envelope seam and
// reports whether the driver conforms. It is engine-agnostic — it exercises only
// what the driver advertises in `meta caps`.
//
//	m-driver-conformance --driver ./dist/m-iris --transport remote
//	m-driver-conformance --driver ./dist/m-ydb  --transport local --json
//
// Connection details are read by the driver from its M_<ENGINE>_* environment,
// inherited here — so set those before running. Exit 0 = conformant, 1 = a check
// failed, 2 = usage.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/vista-cloud-dev/m-driver-sdk/conformance"
)

func main() {
	driver := flag.String("driver", "", "path to the m-<engine> driver binary")
	transport := flag.String("transport", "", "transport to test: local | docker | remote")
	asJSON := flag.Bool("json", false, "emit the report as JSON")
	flag.Parse()

	if *driver == "" || *transport == "" {
		fmt.Fprintln(os.Stderr, "usage: m-driver-conformance --driver <path> --transport <local|docker|remote> [--json]")
		os.Exit(2)
	}

	rep := conformance.Run(context.Background(), conformance.ExecRunner(*driver, *transport), *transport)

	if *asJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(rep)
	} else {
		printText(rep)
	}
	if rep.Fail > 0 {
		os.Exit(1)
	}
}

func printText(rep conformance.Report) {
	title := "m-driver-conformance"
	if rep.Engine != "" {
		title += " — " + rep.Engine
	}
	fmt.Printf("%s (transport %s)\n", title, rep.Transport)
	for _, r := range rep.Results {
		mark := "PASS"
		if !r.Pass {
			mark = "FAIL"
		}
		line := fmt.Sprintf("  [%s] %s", mark, r.Name)
		if !r.Pass && r.Detail != "" {
			line += " — " + r.Detail
		}
		fmt.Println(line)
	}
	fmt.Printf("\n%d passed, %d failed\n", rep.Pass, rep.Fail)
}

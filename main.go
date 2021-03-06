package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	_ "github.com/containerd/cgroups/stats/v1"
	_ "github.com/containerd/cgroups/v2/stats"
	"github.com/containerd/containerd"
	"github.com/containerd/typeurl"
)

func main() {
	addr := os.Getenv("TTRPC_ADDR")
	if addr == "" {
		addr = "/run/containerd/containerd.sock"
	}
	ns := os.Getenv("CONTAINERD_NAMESPACE")
	if ns == "" {
		ns = "k8s.io"
	}

	forceVersion := os.Getenv("CONTAINERD_METRICS_TYPE_URL")
	flag.StringVar(&addr, "addr", addr, "task ttrpc address")
	flag.StringVar(&forceVersion, "metrics-type-url", forceVersion, "force decode with metrics with different type url")

	flag.Parse()

	id := flag.Arg(0)
	if id == "" {
		fmt.Fprintln(os.Stderr, "missing contianer id")
		os.Exit(1)
	}

	client, err := containerd.New(addr, containerd.WithDefaultNamespace(ns))
	if err != nil {
		panic(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	c, err := client.LoadContainer(ctx, flag.Arg(0))
	if err != nil {
		fmt.Fprintf(os.Stderr, "error loading container: %v\n", err)
		os.Exit(2)
	}

	t, err := c.Task(ctx, nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error loading task: %v\n", err)
		os.Exit(2)
	}

	metrics, err := t.Metrics(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error loading metrics: %v\n", err)
		os.Exit(2)
	}

	if forceVersion != "" {
		actual := getActualURL(forceVersion)
		fmt.Fprintln(os.Stderr, "Updating metrics type url from", metrics.Data.TypeUrl, "to", actual)
		metrics.Data.TypeUrl = actual
	}

	anydata, err := typeurl.UnmarshalAny(metrics.Data)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error unmarshalling metrics data for type %q: %v\n", metrics.Data.TypeUrl, err)
		fmt.Fprintln(os.Stderr, metrics.Data)
		os.Exit(3)
	}

	fmt.Println(metrics.Data.TypeUrl)
	if err := json.NewEncoder(os.Stdout).Encode(anydata); err != nil {
		fmt.Fprintf(os.Stderr, "Error marshalling metrics data to json: %v\n", err)
		os.Exit(3)
	}
}

func getActualURL(s string) string {
	switch s {
	case "v1":
		return "io.containerd.cgroups.v1.Metrics"
	case "v2":
		return "io.containerd.cgroups.v2.Metrics"
	default:
		return s
	}
}

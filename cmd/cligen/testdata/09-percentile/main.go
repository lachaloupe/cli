package main

import (
	"context"
	"fmt"
	"math"
	"slices"
	"strconv"
	"strings"

	"github.com/lachaloupe/cli"
)

//go:generate go tool cligen

var CLI = cli.Command{
	Handler: RunPercentile,
	Help:    "Report percentiles from a set of samples",
}

type Args struct {
	// Multiply each sample by this scale factor before reporting.
	//cli:default=1
	Scale float32

	// Percentiles to report.
	//cli:default=50,95,99
	Percentiles []float64

	// Samples to summarize.
	//cli:required
	//cli:arg
	Samples []float64
}

func RunPercentile(ctx context.Context, args Args) error {
	samples := slices.Clone(args.Samples)
	for i := range samples {
		samples[i] *= float64(args.Scale)
	}

	slices.Sort(samples)

	parts := make([]string, 0, len(args.Percentiles))
	for _, p := range args.Percentiles {
		parts = append(parts, fmt.Sprintf("p%s=%s", formatFloat(p), formatFloat(percentile(samples, p))))
	}

	fmt.Printf(
		"percentile scale=%s percentiles=%s samples=%s %s\n",
		formatFloat(float64(args.Scale)),
		formatFloatList(args.Percentiles),
		formatFloatList(samples),
		strings.Join(parts, " "),
	)
	return nil
}

func percentile(samples []float64, p float64) float64 {
	if len(samples) == 0 {
		return 0
	}

	if p <= 0 {
		return samples[0]
	}

	if p >= 100 {
		return samples[len(samples)-1]
	}

	rank := int(math.Ceil((p / 100) * float64(len(samples))))
	return samples[max(0, rank-1)]
}

func formatFloatList(values []float64) string {
	parts := make([]string, len(values))
	for i, value := range values {
		parts[i] = formatFloat(value)
	}
	return strings.Join(parts, ",")
}

func formatFloat(value float64) string {
	return strconv.FormatFloat(value, 'f', -1, 64)
}

func main() {
	CLI.Main()
}

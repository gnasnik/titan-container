package cli

import (
	"fmt"
	"os"

	"github.com/Filecoin-Titan/titan-container/api/types"
	"github.com/Filecoin-Titan/titan-container/lib/tablewriter"
	"github.com/docker/go-units"
	"github.com/urfave/cli/v2"
)

var providerCmds = &cli.Command{
	Name:  "provider",
	Usage: "Manage provider",
	Subcommands: []*cli.Command{
		ProviderList,
	},
}

var ProviderList = &cli.Command{
	Name:  "list",
	Usage: "List providers",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "owner",
			Usage: "owner address",
		},
		&cli.StringFlag{
			Name:  "id",
			Usage: "the provider id",
		},
		&cli.IntFlag{
			Name:  "page",
			Usage: "the page number",
			Value: 1,
		},
		&cli.IntFlag{
			Name:  "size",
			Usage: "the page size",
			Value: 10,
		},
	},
	Action: func(cctx *cli.Context) error {
		api, closer, err := GetManagerAPI(cctx)
		if err != nil {
			return err
		}
		defer closer()

		ctx := ReqContext(cctx)

		tw := tablewriter.New(
			tablewriter.Col("ID"),
			tablewriter.Col("IP"),
			tablewriter.Col("State"),
			tablewriter.Col("ExternalIP"),
			tablewriter.Col("CPUAvail"),
			tablewriter.Col("MemoryAvail"),
			tablewriter.Col("StorageAvail"),
			tablewriter.Col("CreatedTime"),
		)

		opts := &types.GetProviderOption{
			Owner: cctx.String("owner"),
			State: []types.ProviderState{types.ProviderStateOnline, types.ProviderStateOffline, types.ProviderStateAbnormal},
			ID:    types.ProviderID(cctx.String("id")),
			Page:  cctx.Int("page"),
			Size:  cctx.Int("size"),
		}

		providers, err := api.GetProviderList(ctx, opts)
		if err != nil {
			return err
		}

		for _, provider := range providers {
			resource, err := api.GetStatistics(ctx, provider.ID)
			if err != nil {
				continue
			}

			m := map[string]interface{}{
				"ID":           provider.ID,
				"IP":           provider.IP,
				"State":        types.ProviderStateString(provider.State),
				"ExternalIP":   provider.HostURI,
				"CPUAvail":     fmt.Sprintf("%.1f/%.1f", resource.CPUCores.Available, resource.CPUCores.MaxCPUCores),
				"MemoryAvail":  fmt.Sprintf("%s/%s", units.BytesSize(float64(resource.Memory.Available)), units.BytesSize(float64(resource.Memory.MaxMemory))),
				"StorageAvail": fmt.Sprintf("%s/%s", units.BytesSize(float64(resource.Storage.Available)), units.BytesSize(float64(resource.Storage.MaxStorage))),
				"CreateTime":   provider.CreatedAt.Format(defaultDateTimeLayout),
			}
			tw.Write(m)
		}

		tw.Flush(os.Stdout)
		return nil
	},
}

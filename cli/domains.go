package cli

import (
	"github.com/Filecoin-Titan/titan-container/api/types"
	"github.com/Filecoin-Titan/titan-container/lib/tablewriter"
	"github.com/pkg/errors"
	"github.com/urfave/cli/v2"
	"os"
	"strconv"
)

var deploymentDomainCmds = &cli.Command{
	Name:  "domain",
	Usage: "Manager deployment domains",
	Subcommands: []*cli.Command{
		GetDeploymentDomainsCmd,
		AddDeploymentDomainCmd,
		DeleteDeploymentDomainCmd,
	},
}

var GetDeploymentDomainsCmd = &cli.Command{
	Name:  "list",
	Usage: "display deployment domains",
	Flags: []cli.Flag{},
	Action: func(cctx *cli.Context) error {
		if cctx.NArg() != 1 {
			return IncorrectNumArgs(cctx)
		}

		api, closer, err := GetManagerAPI(cctx)
		if err != nil {
			return err
		}
		defer closer()

		ctx := ReqContext(cctx)
		deploymentID := types.DeploymentID(cctx.Args().First())
		if deploymentID == "" {
			return errors.Errorf("deploymentID empty")
		}

		domains, err := api.GetDeploymentDomains(ctx, deploymentID)

		tw := tablewriter.New(
			tablewriter.Col("ID"),
			tablewriter.Col("Hostname"),
			tablewriter.Col("Status"),
		)

		for index, domain := range domains {
			m := map[string]interface{}{
				"ID":       index + 1,
				"Hostname": domain.Host,
				"State":    "OK",
			}
			tw.Write(m)
		}

		return tw.Flush(os.Stdout)
	},
}

var AddDeploymentDomainCmd = &cli.Command{
	Name:  "add",
	Usage: "add new deployment domain configuration",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:     "deployment-id",
			Usage:    "the deployment id",
			Required: true,
		},
	},
	Action: func(cctx *cli.Context) error {
		if cctx.NArg() != 1 {
			return IncorrectNumArgs(cctx)
		}

		api, closer, err := GetManagerAPI(cctx)
		if err != nil {
			return err
		}
		defer closer()

		ctx := ReqContext(cctx)

		deploymentID := types.DeploymentID(cctx.String("deployment-id"))
		if deploymentID == "" {
			return errors.Errorf("deploymentID empty")
		}

		hostname := cctx.Args().First()
		if deploymentID == "" {
			return errors.Errorf("hostname empty")
		}

		err = api.AddDeploymentDomain(ctx, deploymentID, hostname)
		if err != nil {
			return errors.Errorf("add new hostname: %v", err)
		}

		return nil
	},
}

var DeleteDeploymentDomainCmd = &cli.Command{
	Name:  "delete",
	Usage: "delete a deployment domain configuration",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:     "deployment-id",
			Usage:    "the deployment id",
			Required: true,
		},
	},
	Action: func(cctx *cli.Context) error {
		if cctx.NArg() != 1 {
			return IncorrectNumArgs(cctx)
		}

		api, closer, err := GetManagerAPI(cctx)
		if err != nil {
			return err
		}
		defer closer()

		ctx := ReqContext(cctx)

		deploymentID := types.DeploymentID(cctx.String("deployment-id"))
		if deploymentID == "" {
			return errors.Errorf("deploymentID empty")
		}

		index, err := strconv.ParseInt(cctx.Args().First(), 10, 64)
		if err != nil {
			return err
		}

		if index == 1 {
			return errors.Errorf("cannot delete the default domain configuration")
		}

		err = api.DeleteDeploymentDomain(ctx, deploymentID, index)
		if err != nil {
			return errors.Errorf("delete index %d: %v", index, err)
		}

		return nil
	},
}

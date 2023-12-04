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
		GetDomainsCmd,
		AddDomainCmd,
		DeleteDomainCmd,
		ImportCertificateCmd,
	},
}

var GetDomainsCmd = &cli.Command{
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
		if err != nil {
			return err
		}

		tw := tablewriter.New(
			tablewriter.Col("ID"),
			tablewriter.Col("Hostname"),
			tablewriter.Col("State"),
		)

		for index, domain := range domains {
			m := map[string]interface{}{
				"ID":       index + 1,
				"Hostname": domain.Host,
				"State":    domain.State,
			}
			tw.Write(m)
		}

		return tw.Flush(os.Stdout)
	},
}

var AddDomainCmd = &cli.Command{
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

var DeleteDomainCmd = &cli.Command{
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

var ImportCertificateCmd = &cli.Command{
	Name:  "import",
	Usage: "add domain and tls certificate to deployment",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:     "deployment-id",
			Usage:    "the deployment id",
			Required: true,
		},
		&cli.StringFlag{
			Name:     "host",
			Usage:    "hostname",
			Required: true,
		},
		&cli.StringFlag{
			Name:     "key",
			Usage:    "Path to private key associated with given certificate",
			Required: true,
		},
		&cli.StringFlag{
			Name:     "cert",
			Usage:    "Path to PEM encoded public key certificate.",
			Required: true,
		},
	},
	Action: func(cctx *cli.Context) error {
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

		certFile, err := os.ReadFile(cctx.String("cert"))
		if err != nil {
			return err
		}

		certKeyFile, err := os.ReadFile(cctx.String("key"))
		if err != nil {
			return err
		}

		cert := &types.Certificate{
			Host: cctx.String("host"),
			Key:  certKeyFile,
			Cert: certFile,
		}

		err = api.ImportCertificate(ctx, deploymentID, cert)
		if err != nil {
			return errors.Errorf("import certificate: %v", err)
		}

		return nil
	},
}

package cmds

import (
	"fmt"
	"strings"

	"github.com/whiteblock/genesis-cli/pkg/cmds/internal"
	"github.com/whiteblock/genesis-cli/pkg/service"
	"github.com/whiteblock/genesis-cli/pkg/util"

	"github.com/Pallinder/go-randomdata"
	"github.com/spf13/cobra"
)

var runCmd = &cobra.Command{
	Use:   "run <file> [org]",
	Short: "Run a test",
	Long: `Run the tests specified in the given test definition file. If it is your first time running the
	command, you will need to specify the org to deploy to. Subsequent runs will use this value until you specify it
	once again.`,
	Run: func(cmd *cobra.Command, args []string) {
		util.CheckArguments(cmd, args, 1, 2)
		org := ""
		if len(args) > 1 {
			org = args[1]
		}

		data := util.MustReadFile(args[0])
		isCompose, err := cmd.Flags().GetBool("docker-compose")
		if err != nil {
			util.ErrorFatal(err)
		}

		if isCompose {
			data = internal.MustSchemaYAMLFromCompose(data)
		}

		tests, def, err := service.ProcessDefinitionFromBytes(data)
		if err != nil {
			util.ErrorFatal(err)
		}

		var dns []string
		dnsDisabled, err := cmd.Flags().GetBool("no-dns")
		if err != nil {
			util.ErrorFatal(err)
		}

		awaitDisabled, err := cmd.Flags().GetBool("no-await")
		if err != nil {
			util.ErrorFatal(err)
		}

		if !dnsDisabled {
			for range tests {
				dns = append(dns, strings.ToLower(randomdata.SillyName()))
			}
		}
		_, defID, err := service.UploadFiles(args[0], data, org)
		if err != nil {
			util.ErrorFatal(err)
		}

		testIDs, err := service.RunTest(map[string]interface{}{}, org, defID, dns)
		if err != nil {
			util.ErrorFatal(err)
		}

		util.PrintKV(0, "Project", defID)

		for i := range tests {
			util.PrintKV(1, def.Spec.Tests[i].Name, "")
			util.PrintKV(2, "Domains", "")
			if !dnsDisabled {
				for j := range tests[i].ProvisionCommand.Instances {
					util.PrintS(3, fmt.Sprintf("%s-%d.%s", dns[i], j, conf.BiomeDNSZone))

				}
			}
			util.PrintKV(2, "ID", testIDs[i])

		}

		if !awaitDisabled {
			infos := []util.BarInfo{}
			for i := range testIDs {
				infos = append(infos, util.BarInfo{
					Name:  def.Spec.Tests[i].Name,
					Total: tests[i].GuessSteps(),
				})
			}
			if util.IsTTY() {
				channels := map[string]<-chan error{}
				awaiter, bars := util.SetupBars(infos)

				for i := range testIDs {
					channels[def.Spec.Tests[i].Name] = internal.TrackRunStatus(awaiter, bars[i],
						testIDs[i], infos[i])
				}
				awaiter.Wait()
				for test, errChan := range channels {
					err := <-errChan
					if err != nil {
						util.Errorf("%s: %s", test, err.Error())
					}
				}
			} else {
				for i := range testIDs {
					internal.TrackRunStatusNoTTY(testIDs[i], infos[i].Total)
				}
			}
		}

	},
}

func init() {

	runCmd.Flags().BoolP("no-dns", "d", false, "disable assigning a DNS name to your deployment")
	runCmd.Flags().BoolP("docker-compose", "c", false, "deploy from a docker compose file")
	runCmd.Flags().BoolP("no-await", "a", false, "don't wait for completion, exit immediately after sending test")
	rootCmd.AddCommand(runCmd)
}

package main

import (
	"fmt"
	_ "net/http/pprof"
	"os"

	"github.com/Financial-Times/base-ft-rw-app-go"
	"github.com/Financial-Times/brands-rw-neo4j/brands"
	"github.com/Financial-Times/go-fthealth/v1a"
	"github.com/Financial-Times/neo-utils-go"
	log "github.com/Sirupsen/logrus"
	"github.com/jawher/mow.cli"
	"github.com/jmcvetta/neoism"
)

func main() {
	log.SetLevel(log.InfoLevel)
	log.Infof("Application started with args %s", os.Args)

	app := cli.App("people-rw-neo4j", "A RESTful API for managing People in neo4j")
	neoURL := app.StringOpt("neo-url", "http://localhost:7474/db/data", "neo4j endpoint URL")
	port := app.IntOpt("port", 8080, "Port to listen on")
	batchSize := app.IntOpt("batchSize", 1024, "Maximum number of statements to execute per batch")
	graphiteTCPAddress := app.StringOpt("graphiteTCPAddress", "",
		"Graphite TCP address, e.g. graphite.ft.com:2003. Leave as default if you do NOT want to output to graphite (e.g. if running locally)")
	graphitePrefix := app.StringOpt("graphitePrefix", "",
		"Prefix to use. Should start with content, include the environment, and the host name. e.g. content.test.people.rw.neo4j.ftaps58938-law1a-eu-t")
	logMetrics := app.BoolOpt("logMetrics", false, "Whether to log metrics. Set to true if running locally and you want metrics output")

	app.Action = func() {
		db, err := neoism.Connect(*neoURL)
		if err != nil {
			log.Errorf("Could not connect to neo4j, error=[%s]\n", err)
		}

		batchRunner := neoutils.NewBatchCypherRunner(neoutils.StringerDb{db}, *batchSize)
		brandsDriver := brands.NewCypherBrandsService(batchRunner, db)

		baseftrwapp.OutputMetricsIfRequired(*graphiteTCPAddress, *graphitePrefix, *logMetrics)

		engs := map[string]baseftrwapp.Service{
			"brands": brandsDriver,
		}

		var checks []v1a.Check
		for _, e := range engs {
			checks = append(checks, makeCheck(e, batchRunner))
		}

		baseftrwapp.RunServer(engs,
			v1a.Handler("ft-brands_rw_neo4j ServiceModule", "Writes 'brands' to Neo4j, usually as part of a bulk upload done on a schedule", checks...),
			*port)
	}

	app.Run(os.Args)
}

func makeCheck(service baseftrwapp.Service, cr neoutils.CypherRunner) v1a.Check {
	return v1a.Check{
		BusinessImpact:   "Cannot read/write people via this writer",
		Name:             "Check connectivity to Neo4j - neoUrl is a parameter in hieradata for this service",
		PanicGuide:       "TODO - write panic guide",
		Severity:         1,
		TechnicalSummary: fmt.Sprintf("Cannot connect to Neo4j instance %s with at least one person loaded in it", cr),
		Checker:          func() (string, error) { return "", service.Check() },
	}
}
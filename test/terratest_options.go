package test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/gruntwork-io/terratest/modules/terraform"
)

func createExampleTerraformOptions(
	t *testing.T,
	uniqueID,
	project string,
	region string,
	templatePath string,
) *terraform.Options {
	repoName := strings.ToLower(fmt.Sprintf("sample-docker-app-%s", uniqueID))
	serviceName := strings.ToLower(fmt.Sprintf("sample-docker-service-%s", uniqueID))

	terraformVars := map[string]interface{}{
		"location":        region,
		"project":         project,
		"gcr_region":      lookupMultiRegion(region),
		"repository_name": repoName,
		"service_name":    serviceName,
	}

	terratestOptions := terraform.Options{
		TerraformDir: templatePath,
		Vars:         terraformVars,
	}

	return &terratestOptions
}

func createExampleWithMysqlTerraformOptions(
	t *testing.T,
	uniqueID,
	project string,
	region string,
	templatePath string,
) *terraform.Options {
	repoName := strings.ToLower(fmt.Sprintf("sample-docker-app-%s", uniqueID))
	serviceName := strings.ToLower(fmt.Sprintf("sample-docker-service-%s", uniqueID))

	terraformVars := map[string]interface{}{
		"location":        region,
		"project":         project,
		"gcr_region":      lookupMultiRegion(region),
		"repository_name": repoName,
		"service_name":    serviceName,
		"deploy_db":       true,
	}

	terratestOptions := terraform.Options{
		TerraformDir: templatePath,
		Vars:         terraformVars,
	}

	return &terratestOptions
}

// lookupMultiRegion returns the appropriate multi-region depending on the GCP region passed in.
// https://cloud.google.com/storage/docs/locations#location-mr
func lookupMultiRegion(region string) string {
	parts := strings.Split(region, "-")

	switch mr := parts[0]; mr {
	case "europe":
		return "eu"
	case "asia":
		return "asia"
	default:
		return "us"
	}
}

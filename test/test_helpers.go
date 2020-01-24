package test

import (
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/gruntwork-io/terratest/modules/gcp"
	"github.com/gruntwork-io/terratest/modules/logger"
	"github.com/gruntwork-io/terratest/modules/retry"
	cloudbuildpb "google.golang.org/genproto/googleapis/devtools/cloudbuild/v1"
)

func verifyBuildWasSuccessful(t *testing.T, projectID string, triggerID string) string {
	statusMsg := fmt.Sprintf("Wait for build to complete.")
	retries := 30
	sleepBetweenRetries := 20 * time.Second

	successfulBuildID, err := retry.DoWithRetryE(
		t,
		statusMsg,
		retries,
		sleepBetweenRetries,
		func() (string, error) {
			builds := gcp.GetBuildsForTrigger(t, projectID, triggerID)

			if len(builds) == 0 {
				return "", errors.New("Build hasn't been triggered")
			}

			// assume the first build returned is the one we triggered.
			buildID := builds[0].GetId()
			build, err := gcp.GetBuildE(t, projectID, buildID)
			if err != nil {
				return "", err
			}

			if build.GetStatus() == cloudbuildpb.Build_QUEUED {
				return "", errors.New("Build is queued")
			}

			if build.GetStatus() == cloudbuildpb.Build_WORKING {
				return "", errors.New("Build is executing")
			}

			if build.GetStatus() != cloudbuildpb.Build_SUCCESS {
				return "", errors.New("Build is not successful")
			}

			return build.GetId(), nil
		},
	)
	if err != nil {
		logger.Logf(t, "Error waiting for the build to complete: %s", err)
		t.Fatal(err)
	}
	logger.Logf(t, "Build was successful")
	return successfulBuildID
}

func cleanName(originalName string) string {
	parts := strings.Split(originalName, "/")
	return parts[len(parts)-1]
}

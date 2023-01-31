package test

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/gruntwork-io/terratest/modules/files"
	"github.com/gruntwork-io/terratest/modules/gcp"
	"github.com/gruntwork-io/terratest/modules/logger"
	"github.com/gruntwork-io/terratest/modules/random"
	"github.com/gruntwork-io/terratest/modules/shell"
	"github.com/gruntwork-io/terratest/modules/terraform"
	test_structure "github.com/gruntwork-io/terratest/modules/test-structure"
	"github.com/stretchr/testify/require"
)

// The GitHub repo name.
const GitHubRepoName = "sample-app-docker"

// The GitHub repo url.
const GitHubRepoURL = "https://github.com/robmorgan/sample-node-app.git"

// The Git branch to check out and push commits to.
const GitHubRepoBranch = "main"

// The following test ensures the example works as expected. It performs the following test logic:
//
// 1. Clones a sample Node app to its own folder.
// 2. Deploys all of the example resources using Terraform.
// 4. Adds a Git remote to the sample app repo that points to the Cloud Source Repository.
// 5. Commits a test file and pushes it to trigger a build.
// 6. Polls the Cloud Build API to check if the build was successful.
// 7. Cleans up all of the resources using Terraform destroy.
// 8. Ensures all of the GCR images are removed.
func TestExample(t *testing.T) {
	t.Parallel()

	// Uncomment any of the following to skip that section during the test
	//os.Setenv("SKIP_clone_sample_app", "true")
	//os.Setenv("SKIP_create_test_copy_of_examples", "true")
	//os.Setenv("SKIP_create_terratest_options", "true")
	//os.Setenv("SKIP_terraform_apply", "true")
	//os.Setenv("SKIP_trigger_build", "true")
	//os.Setenv("SKIP_wait_for_build", "true")
	//os.Setenv("SKIP_cleanup", "true")

	// Create a directory path that won't conflict
	workingDir := filepath.Join(".", "stages", t.Name())

	test_structure.RunTestStage(t, "clone_sample_app", func() {
		tmpDir, err := ioutil.TempDir("", "repos")
		require.NoError(t, err)

		// Inside of the temp folder, we create a subfolder that preserves the name of the folder we're copying from.
		absFolderPath, err := filepath.Abs(GitHubRepoName)
		require.NoError(t, err)

		folderName := filepath.Base(absFolderPath)
		destFolder := filepath.Join(tmpDir, folderName)

		err = os.MkdirAll(destFolder, 0777)
		require.NoError(t, err)

		logger.Logf(t, "shallow cloning git repo %s to tmp folder %s\n", GitHubRepoURL, destFolder)
		cmd := exec.Command("git", "clone", GitHubRepoURL, "--branch", GitHubRepoBranch, "--single-branch", destFolder)
		_, err = cmd.Output()
		require.NoError(t, err)
		test_structure.SaveString(t, workingDir, "repoPath", destFolder)
	})

	test_structure.RunTestStage(t, "create_test_copy_of_examples", func() {
		rootFolder := ".."
		tmpTestFolder, err := files.CopyTerraformFolderToTemp(rootFolder, cleanName(t.Name()))
		require.NoError(t, err)

		// Log temp folder so we can see it
		logger.Logf(t, "Copied terraform folder %s to %s", rootFolder, tmpTestFolder)
		logger.Logf(t, "path to test folder %s\n", tmpTestFolder)
		terraformModulePath := filepath.Join(tmpTestFolder)
		test_structure.SaveString(t, workingDir, "exampleTerraformModulePath", terraformModulePath)
	})

	test_structure.RunTestStage(t, "create_terratest_options", func() {
		exampleTerraformModulePath := test_structure.LoadString(t, workingDir, "exampleTerraformModulePath")
		uniqueID := random.UniqueId()
		project := gcp.GetGoogleProjectIDFromEnvVar(t)
		allowedRegions := []string{"asia-northeast1", "europe-west1", "us-central1", "us-east1"} // cloud run has limited availability
		region := gcp.GetRandomRegion(t, project, allowedRegions, nil)
		exampleTerratestOptions := createExampleTerraformOptions(t, uniqueID, project, region, exampleTerraformModulePath)
		test_structure.SaveString(t, workingDir, "uniqueID", uniqueID)
		test_structure.SaveString(t, workingDir, "project", project)
		test_structure.SaveString(t, workingDir, "region", region)
		test_structure.SaveTerraformOptions(t, workingDir, exampleTerratestOptions)
	})

	defer test_structure.RunTestStage(t, "cleanup", func() {
		project := test_structure.LoadString(t, workingDir, "project")
		buildID := test_structure.LoadString(t, workingDir, "buildID")

		build := gcp.GetBuild(t, project, buildID)
		for _, image := range build.GetImages() {
			gcp.DeleteGCRImageRef(t, image)
		}

		exampleTerratestOptions := test_structure.LoadTerraformOptions(t, workingDir)
		terraform.Destroy(t, exampleTerratestOptions)
	})

	test_structure.RunTestStage(t, "terraform_apply", func() {
		exampleTerratestOptions := test_structure.LoadTerraformOptions(t, workingDir)
		terraform.InitAndApply(t, exampleTerratestOptions)
	})

	test_structure.RunTestStage(t, "trigger_build", func() {
		exampleTerratestOptions := test_structure.LoadTerraformOptions(t, workingDir)
		project := test_structure.LoadString(t, workingDir, "project")
		repoName := exampleTerratestOptions.Vars["repository_name"].(string)
		repoPath := test_structure.LoadString(t, workingDir, "repoPath")

		// add the cloud source repository as a git remote if it doesn't already exist
		// `git remote add google https://source.developers.google.com/p/[PROJECT_ID]/r/[REPO_NAME]`
		if _, err := os.Stat(fmt.Sprintf("%s/.git/refs/remotes/google", repoPath)); os.IsNotExist(err) {
			cmd := shell.Command{
				Command:    "git",
				Args:       []string{"remote", "add", "google", fmt.Sprintf("https://source.developers.google.com/p/%s/r/%s", project, repoName)},
				WorkingDir: repoPath,
			}

			shell.RunCommand(t, cmd)
		}

		// write a test file
		date := []byte(fmt.Sprintf("%s\n", time.Now().String()))
		testFile := fmt.Sprintf("%s/auto-committed.txt", repoPath)
		err := ioutil.WriteFile(testFile, date, 0644)
		require.NoError(t, err)

		// commit and push
		cmd2 := shell.Command{
			Command:    "git-add-commit-push",
			Args:       []string{"--remote-name", "google", "--path", testFile, "--message", "triggering a build", "--skip-git-pull", "--skip-ci-flag", ""},
			WorkingDir: repoPath,
		}

		shell.RunCommand(t, cmd2)
	})

	test_structure.RunTestStage(t, "wait_for_build", func() {
		exampleTerratestOptions := test_structure.LoadTerraformOptions(t, workingDir)
		triggerID := terraform.Output(t, exampleTerratestOptions, "trigger_id")
		project := test_structure.LoadString(t, workingDir, "project")
		buildID := verifyBuildWasSuccessful(t, project, triggerID)
		test_structure.SaveString(t, workingDir, "buildID", buildID)
	})
}

func TestExampleWithMysql(t *testing.T) {
	t.Parallel()

	// Uncomment any of the following to skip that section during the test
	//os.Setenv("SKIP_clone_sample_app", "true")
	//os.Setenv("SKIP_create_test_copy_of_examples", "true")
	//os.Setenv("SKIP_create_terratest_options", "true")
	//os.Setenv("SKIP_terraform_apply", "true")
	//os.Setenv("SKIP_trigger_build", "true")
	//os.Setenv("SKIP_wait_for_build", "true")
	//os.Setenv("SKIP_cleanup", "true")

	// Create a directory path that won't conflict
	workingDir := filepath.Join(".", "stages", t.Name())

	test_structure.RunTestStage(t, "clone_sample_app", func() {
		tmpDir, err := ioutil.TempDir("", "repos")
		require.NoError(t, err)

		// Inside of the temp folder, we create a subfolder that preserves the name of the folder we're copying from.
		absFolderPath, err := filepath.Abs(GitHubRepoName)
		require.NoError(t, err)

		folderName := filepath.Base(absFolderPath)
		destFolder := filepath.Join(tmpDir, folderName)

		err = os.MkdirAll(destFolder, 0777)
		require.NoError(t, err)

		logger.Logf(t, "shallow cloning git repo %s to tmp folder %s\n", GitHubRepoURL, destFolder)
		cmd := exec.Command("git", "clone", GitHubRepoURL, "--branch", GitHubRepoBranch, "--single-branch", destFolder)
		_, err = cmd.Output()
		require.NoError(t, err)
		test_structure.SaveString(t, workingDir, "repoPath", destFolder)
	})

	test_structure.RunTestStage(t, "create_test_copy_of_examples", func() {
		rootFolder := ".."
		tmpTestFolder, err := files.CopyTerraformFolderToTemp(rootFolder, cleanName(t.Name()))
		require.NoError(t, err)

		// Log temp folder so we can see it
		logger.Logf(t, "Copied terraform folder %s to %s", rootFolder, tmpTestFolder)
		logger.Logf(t, "path to test folder %s\n", tmpTestFolder)
		terraformModulePath := filepath.Join(tmpTestFolder)
		test_structure.SaveString(t, workingDir, "exampleTerraformModulePath", terraformModulePath)
	})

	test_structure.RunTestStage(t, "create_terratest_options", func() {
		exampleTerraformModulePath := test_structure.LoadString(t, workingDir, "exampleTerraformModulePath")
		uniqueID := random.UniqueId()
		project := gcp.GetGoogleProjectIDFromEnvVar(t)
		allowedRegions := []string{"asia-northeast1", "europe-west1", "us-central1", "us-east1"} // cloud run has limited availability
		region := gcp.GetRandomRegion(t, project, allowedRegions, nil)
		exampleTerratestOptions := createExampleWithMysqlTerraformOptions(t, uniqueID, project, region, exampleTerraformModulePath)
		test_structure.SaveString(t, workingDir, "uniqueID", uniqueID)
		test_structure.SaveString(t, workingDir, "project", project)
		test_structure.SaveString(t, workingDir, "region", region)
		test_structure.SaveTerraformOptions(t, workingDir, exampleTerratestOptions)
	})

	defer test_structure.RunTestStage(t, "cleanup", func() {
		project := test_structure.LoadString(t, workingDir, "project")
		buildID := test_structure.LoadString(t, workingDir, "buildID")

		build := gcp.GetBuild(t, project, buildID)
		for _, image := range build.GetImages() {
			gcp.DeleteGCRImageRef(t, image)
		}

		exampleTerratestOptions := test_structure.LoadTerraformOptions(t, workingDir)
		terraform.Destroy(t, exampleTerratestOptions)
	})

	test_structure.RunTestStage(t, "terraform_apply", func() {
		exampleTerratestOptions := test_structure.LoadTerraformOptions(t, workingDir)
		terraform.InitAndApply(t, exampleTerratestOptions)
	})

	test_structure.RunTestStage(t, "trigger_build", func() {
		exampleTerratestOptions := test_structure.LoadTerraformOptions(t, workingDir)
		project := test_structure.LoadString(t, workingDir, "project")
		repoName := exampleTerratestOptions.Vars["repository_name"].(string)
		repoPath := test_structure.LoadString(t, workingDir, "repoPath")

		// add the cloud source repository as a git remote if it doesn't already exist
		// `git remote add google https://source.developers.google.com/p/[PROJECT_ID]/r/[REPO_NAME]`
		if _, err := os.Stat(fmt.Sprintf("%s/.git/refs/remotes/google", repoPath)); os.IsNotExist(err) {
			cmd := shell.Command{
				Command:    "git",
				Args:       []string{"remote", "add", "google", fmt.Sprintf("https://source.developers.google.com/p/%s/r/%s", project, repoName)},
				WorkingDir: repoPath,
			}

			shell.RunCommand(t, cmd)
		}

		// write a test file
		date := []byte(fmt.Sprintf("%s\n", time.Now().String()))
		testFile := fmt.Sprintf("%s/auto-committed.txt", repoPath)
		err := ioutil.WriteFile(testFile, date, 0644)
		require.NoError(t, err)

		// commit and push
		cmd2 := shell.Command{
			Command:    "git-add-commit-push",
			Args:       []string{"--remote-name", "google", "--path", testFile, "--message", "triggering a build", "--skip-git-pull", "--skip-ci-flag", ""},
			WorkingDir: repoPath,
		}

		shell.RunCommand(t, cmd2)
	})

	test_structure.RunTestStage(t, "wait_for_build", func() {
		exampleTerratestOptions := test_structure.LoadTerraformOptions(t, workingDir)
		triggerID := terraform.Output(t, exampleTerratestOptions, "trigger_id")
		project := test_structure.LoadString(t, workingDir, "project")
		buildID := verifyBuildWasSuccessful(t, project, triggerID)
		test_structure.SaveString(t, workingDir, "buildID", buildID)
	})
}

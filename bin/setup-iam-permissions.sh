#!/bin/bash

set -e

if [[ -z "${GOOGLE_CLOUD_PROJECT}" ]]; then
  echo "Please make sure GOOGLE_CLOUD_PROJECT is defined before running this script."
  exit 1
fi

echo "Discovering Project ID for project $GOOGLE_CLOUD_PROJECT..."
PROJECT_NUM=$(gcloud projects describe $GOOGLE_CLOUD_PROJECT --format='value(projectNumber)')

echo "Got project number: ${PROJECT_NUM}"

echo "Applying IAM Policy Binding..."
gcloud iam service-accounts add-iam-policy-binding \
  ${PROJECT_NUM}-compute@developer.gserviceaccount.com \
  --member=serviceAccount:${PROJECT_NUM}@cloudbuild.gserviceaccount.com \
  --role=roles/iam.serviceAccountUser \
  --project=${GOOGLE_CLOUD_PROJECT}

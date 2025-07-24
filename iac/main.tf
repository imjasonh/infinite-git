terraform {
  required_providers {
    google = {      source  = "hashicorp/google"    }
    ko = {      source  = "ko-build/ko"    }
  }
}

variable "project_id" {
  description = "The GCP project ID"
  type        = string
}

variable "regions" {
  description = "The GCP regions to deploy to"
  type        = list(string)
  default     = ["us-east4"]
}

provider "google" {
  project = var.project_id
  region  = var.regions[0]
}

provider "ko" {
  repo = "${var.regions[0]}-docker.pkg.dev/${var.project_id}/infinite-git/app"
}

# Create Artifact Registry repository
resource "google_artifact_registry_repository" "repo" {
  project       = var.project_id
  location      = var.regions[0]
  repository_id = "infinite-git"
  format        = "DOCKER"
  description   = "Docker repository for Cloud Run deployments"
}

# Set up networking
module "networking" {
  source = "github.com/chainguard-dev/terraform-infra-common//modules/networking"

  project_id = var.project_id
  name       = "infinite-git"
  regions    = var.regions
}

module "infinite-git" {
  source = "github.com/chainguard-dev/terraform-infra-common//modules/regional-go-service"

  project_id = var.project_id
  name       = "infinite-git"
  regions    = module.networking.regional-networks

  service_account = google_service_account.infinite_git.email
  
  depends_on = [google_artifact_registry_repository.repo]
  
  containers = {
    "infinite-git" = {
      source = {
        working_dir = ".."
        importpath  = "."
      }
      ports = [{
        container_port = 8080
      }]
      env = [{
        name  = "REPO_PATH"
        value = "/tmp/infinite-git-repo"
      }]
    }
  }

  notification_channels = []
  require_squad        = false

  # Enable public access
  ingress = "INGRESS_TRAFFIC_ALL"
}

resource "google_service_account" "infinite_git" {
  project      = var.project_id
  account_id   = "infinite-git"
  display_name = "Infinite Git Service Account"
  description  = "Service account for the infinite-git Cloud Run service"
}

output "uris" {
  description = "The URIs of the deployed service by region"
  value       = module.infinite-git.uris
}

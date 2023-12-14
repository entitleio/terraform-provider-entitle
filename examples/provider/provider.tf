terraform {
  required_providers {
    entitle = {
      source = "entitle-io/entitle"
    }
  }
}

provider "entitle" {
  endpoint = "https://api.entitle.io"
  api_key  = "PUT_YOUR_TOKEN"
}

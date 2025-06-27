terraform {
  required_providers {
    entitle = {
      source = "entitleio/entitle"
    }
  }
}

provider "entitle" {
  endpoint = "https://api.entitle.io"
  api_key  = "PUT_YOUR_TOKEN"
}

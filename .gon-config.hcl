source    = ["./dist/terraform-exporter_darwin_all/terraform-exporter"]
bundle_id = "io.gideaworx.terraform-exporter"

apple_id {
  username = "ghiloni@gmail.com"
  password = "@env:AC_PASSWORD"
}

sign {
  application_identity = "8210B55D14042D8C8F65B2783A0E52250D28A3B7"
}

dmg {
  output_path = "./dist/terraform-exporter-macos.dmg"
  volume_name = "Terraform Exporter"
}

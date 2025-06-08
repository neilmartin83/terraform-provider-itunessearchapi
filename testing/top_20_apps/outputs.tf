
output "saved_icons" {
  value = [for icon in local_file.app_icons : icon.filename]
}

# Query an existing volume by ID
data "emma_volume" "existing" {
  id = "12345"
}

# Use the volume data in other resources
output "volume_size" {
  value = data.emma_volume.existing.volume_gb
}

output "volume_status" {
  value = data.emma_volume.existing.status
}

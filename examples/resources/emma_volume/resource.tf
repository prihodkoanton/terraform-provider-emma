# Basic volume creation
resource "emma_volume" "data_volume" {
  data_center_id = data.emma_data_center.aws.id
  volume_gb      = 100
  volume_type    = "ssd"
  name           = "my-data-volume"
}

# Volume with attachment to a VM
resource "emma_volume" "attached_volume" {
  data_center_id = data.emma_data_center.aws.id
  volume_gb      = 50
  volume_type    = "ssd"
  name           = "attached-volume"
  attached_to_id = emma_vm.my_vm.id
}

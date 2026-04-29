# Query available volume configurations for a data center
data "emma_volume_configurations" "aws_configs" {
  data_center_id = data.emma_data_center.aws.id
}

# Display available volume types and their specifications
output "available_volume_types" {
  value = data.emma_volume_configurations.aws_configs.configurations
}

# Use configuration data to create a volume with valid parameters
resource "emma_volume" "configured_volume" {
  data_center_id = data.emma_data_center.aws.id
  volume_gb      = 100
  volume_type    = data.emma_volume_configurations.aws_configs.configurations[0].volume_type
  name           = "configured-volume"
}

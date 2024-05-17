output "container_registry_id" {
  value = yandex_container_registry.cr-hse.id
}

output "sa_builder_id" {
  value = yandex_iam_service_account.sa-builder.id
}

output "sa_grader_id" {
  value = yandex_iam_service_account.sa-grader.id
}

output "nat_public_ip" {
  value = yandex_compute_instance.nat-instance.network_interface.0.nat_ip_address
}

output "graders_group_id" {
  value = yandex_compute_instance_group.graders.id
}

output "graders_group_internal_ips" {
  value = yandex_compute_instance_group.graders.instances.*.network_interface.0.ip_address
}

output "builder_id" {
  value = yandex_compute_instance.builder.id
}

output "builder_internal_ip" {
  value = yandex_compute_instance.builder.network_interface.0.ip_address
}

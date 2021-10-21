output "ip" {
 value = aws_instance.default.public_ip
}

output "username" {
 value = var.user
}

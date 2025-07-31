data "opsecret_reference" "secret_reference" {
  id = "op://vault-name/item-name/section-name/field-name"
}

resource "whatever" "some_resource" {
  attribute = data.opsecret_reference.secret_reference.value
}
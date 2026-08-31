# Test-only SSH material for the acceptance harness.
# Never reuse these keys outside docker-compose.test.yml / test/acceptance.
#
# Files:
#   id_ed25519[.pub]             — client key A (worker)
#   id_ed25519_b[.pub]           — client key B (worker-b)
#   ssh_host_ed25519_key[.pub]   — fixed worker host key (known_hosts is derived)
#   known_hosts                  — StrictHostKeyChecking entries for worker, worker-b, localhost

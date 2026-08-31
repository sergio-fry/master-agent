# Acceptance-only sshd worker: key auth, bash, stub-friendly shell.
# No Cursor, backlog, or MCP — remote commands are touch/echo/exit/sleep.
FROM alpine:3.22

RUN apk add --no-cache openssh-server bash \
	&& adduser -D -h /home/worker -s /bin/bash worker \
	&& passwd -u worker \
	&& mkdir -p /var/run/sshd /home/worker/.ssh /home/worker/workspace \
	&& chown -R worker:worker /home/worker

COPY test/fixtures/ssh/ssh_host_ed25519_key /etc/ssh/ssh_host_ed25519_key
COPY test/fixtures/ssh/id_ed25519.pub /home/worker/.ssh/authorized_keys
COPY test/worker/sshd_config /etc/ssh/sshd_config
COPY test/worker/entrypoint.sh /entrypoint.sh

RUN chmod 600 /etc/ssh/ssh_host_ed25519_key \
	&& chmod 700 /home/worker/.ssh \
	&& chmod 600 /home/worker/.ssh/authorized_keys \
	&& chmod 755 /entrypoint.sh \
	&& chown -R worker:worker /home/worker

EXPOSE 22
CMD ["/entrypoint.sh"]

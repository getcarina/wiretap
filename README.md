wiretap
=======

wiretap is a small web server that listens out for Docker Hub webhooks. Once it receives a JSON payload, it matches existing containers and restarts them gracefully if required. It does this by sending a ``SIGTERM`` signal to the remote API's kill operation, allowing the container to handle remaining connections before being closed.

This library is meant as an initial foray into continuous deployment on Docker Swarm, and should be used cautiously. Some parts have been adapted from [watchtower](https://github.com/getcarina/watchtower).

## Run the container

Generate a secret or use your own:

```bash
export SECRET=$(openssl rand -base64 30)
```

Then run the container:

```bash
docker run --detach \
  --name wiretap \
  --publish 8000:8000 \
  --volumes-from swarm-data \
  --env TOKEN=$SECRET \
  --env DOCKER_CERT_PATH=/etc/docker/cert.pem \
  --env DOCKER_CA_CERT_PATH=/etc/docker/ca.pem \
  --env DOCKER_KEY_PATH=/etc/docker/key.pem \
  carina/wiretap
```

The web server will now be listening for POST requests. To test, you can run

```bash
curl -i -X POST "$(docker port wiretap 8000)/listen?token=$SECRET" --data "@webhook.json"
```

where ``webhook.json`` contains the JSON payload documented [here](https://docs.docker.com/docker-hub/webhooks/).

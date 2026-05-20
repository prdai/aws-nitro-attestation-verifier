# AWS Nitro Attestation Verifier

aws nitro enclaves remote attestation verifier with enclave attestation generation, parent host relay, and client side verification

---

ok so what we need to do is this right- we need to have a external client (can be anything, the data owners, or even our own system), we should be able to call the AWS ECS and then for it to redirect the communication to the relevant nitro enclave (we need to see how we can do this- is it like a id based thing or in what way can be do this communication). 

after the request is reached to by the nitro encalve then we need to some how make a attestation document within there and then return it to the ECS and back to the client, and afterwards the client can verify and continue on communication between the client and the nitro enclave instance with encryption.

![./assets/architecture-diagram.svg](./assets/architecture-diagram.svg)

references:
- https://docs.aws.amazon.com/enclaves/latest/user/nitro-enclave.html
- https://github.com/aws-samples/sample-nitro-enclaves-attestation
- https://medium.com/@mdlayher/linux-vm-sockets-in-go-ea11768e9e67

## Current workflow

The workflow is split across three binaries because each side has a different
trust role.

1. `client/cmd/request` runs outside AWS. It creates or accepts a nonce, calls
   the EC2 relay, and verifies the returned COSE_Sign1 attestation document
   against the AWS Nitro Enclaves root certificate.
2. `ec2/cmd/server` runs on the parent EC2 host. It exposes public HTTP on
   `GET /attestation`, forwards the client nonce to the enclave over vsock
   using `ENCLAVE_CID` and `ENCLAVE_PORT`, and returns the enclave document to
   the external client. EC2 is only a relay here; it is not trusted by the
   verifier.
3. `nitro-enclave/cmd/attestation` runs inside the Nitro Enclave. It listens
   on a vsock port, decodes the client nonce, asks the Nitro Security Module at
   `/dev/nsm` for a fresh attestation document, and returns that document to
   the parent EC2 process.

The verifier checks the certificate chain, the COSE signature over the signed
payload, and caller-supplied expectations such as nonce and PCR values. The
nonce matters because it binds the attestation response to the client's fresh
request instead of accepting a replayed document.

## Deployed enclave workflow

Provision the parent EC2 host first:

```sh
make infra-init
make infra-deploy
```

On the parent EC2 host, build the enclave Docker image, convert it to an EIF,
and start the enclave:

```sh
make enclave-docker-build
make enclave-eif-build
make enclave-run
make enclave-describe
```

Run the parent EC2 relay on the same instance:

```sh
make ec2-server-run
```

Run the external client from your machine:

```sh
make nitro-root-cert
cd client
go run ./cmd/request \
  -url http://EC2_PUBLIC_IP:8080/attestation \
  -root-cert ../.cache/aws-nitro-root/AWS_NitroEnclaves_Root-G1.pem
```

Stop the enclave and destroy the parent host when finished:

```sh
make enclave-stop
make infra-destroy
```

The Docker image is only the build input. `nitro-cli build-enclave` converts it
into an EIF, and `nitro-cli run-enclave` runs that EIF as the isolated Nitro
Enclave. The enclave itself has no public network path.

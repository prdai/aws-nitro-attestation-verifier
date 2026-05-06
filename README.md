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

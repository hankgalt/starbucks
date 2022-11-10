CONFIG_PATH=${HOME}/.certs/

.PHONY: init
init:
		mkdir -p ${CONFIG_PATH}

.PHONY: gencacert
gencacert:
		cfssl gencert -initca templates/ca-csr.json | cfssljson -bare ca -config=templates/ca-config.json
		mv *.pem *.csr ${CONFIG_PATH}

.PHONY: genscert
genscert:
		cfssl gencert -ca=${CONFIG_PATH}ca.pem -ca-key=${CONFIG_PATH}ca-key.pem -profile=server templates/server-csr.json | cfssljson -bare server
		mv *.pem *.csr ${CONFIG_PATH}

.PHONY: genccert
genccert:
		cfssl gencert -ca=${CONFIG_PATH}ca.pem -ca-key=${CONFIG_PATH}ca-key.pem -profile=client templates/client-csr.json | cfssljson -bare client
		mv *.pem *.csr ${CONFIG_PATH}

.PHONY: genrccert
genrccert:
		cfssl gencert -ca=${CONFIG_PATH}ca.pem -ca-key=${CONFIG_PATH}ca-key.pem -profile=client -cn="root" templates/client-csr.json | cfssljson -bare root-client
		mv *.pem *.csr ${CONFIG_PATH}

.PHONY: gennccert
gennccert:
		cfssl gencert -ca=${CONFIG_PATH}ca.pem -ca-key=${CONFIG_PATH}ca-key.pem -profile=client -cn="nobody" templates/client-csr.json | cfssljson -bare nobody-client
		mv *.pem *.csr ${CONFIG_PATH}

.PHONY: cppolicy
cppolicy:
	cp templates/model.conf $(CONFIG_PATH)/model.conf
	cp templates/policy.csv $(CONFIG_PATH)/policy.csv

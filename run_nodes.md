./build/axia \
--genesis=./genesis/localhost_genesis.json \
--http-host=127.0.0.1 \
--http-port=9650 \
--staking-port=9651 \
--db-dir=db/node1 \
--network-id=7890 \
--staking-tls-cert-file=$(pwd)/staking/local/staker1.crt \
--staking-tls-key-file=$(pwd)/staking/local/staker1.key

./build/axia
--genesis=./genesis/localhost_genesis.json
--http-host=127.0.0.1
--http-port=9652
--staking-port=9653
--db-dir=db/node2
--network-id=7890
--staking-tls-cert-file=$(pwd)/staking/local/staker2.crt
--staking-tls-key-file=$(pwd)/staking/local/staker2.key

./build/axia
--genesis=./genesis/localhost_genesis.json
--http-host=127.0.0.1
--http-port=9654
--staking-port=9655
--db-dir=db/node3
--network-id=7890
--staking-tls-cert-file=$(pwd)/staking/local/staker3.crt
--staking-tls-key-file=$(pwd)/staking/local/staker3.key

./build/axia
--genesis=./genesis/localhost_genesis.json
--http-host=127.0.0.1
--http-port=9656
--staking-port=9657
--db-dir=db/node4
--network-id=7890
--staking-tls-cert-file=$(pwd)/staking/local/staker4.crt
--staking-tls-key-file=$(pwd)/staking/local/staker4.key

Production
./build/axia \
--genesis=genesis.json \
--public-ip=18.222.205.99 \ 
--http-host=0.0.0.0 \
--http-port=9650 \
--staking-port=9651 \
--db-dir=db/node1 \
--network-id=5678 \
--staking-tls-cert-file=$(pwd)/staking/local/staker1.crt \ --staking-tls-key-file=$(pwd)/staking/local/staker1.key
Sample
./build/axia
--genesis=./genesis/genesis.json
--http-host=0.0.0.0
--http-port=9650
--staking-port=9651
--db-dir=db/node1
--network-id=5678
--staking-tls-cert-file=$(pwd)/staking/local/staker1.crt
--staking-tls-key-file=$(pwd)/staking/local/staker1.key

./build/axia
--genesis=./genesis/genesis.json
--http-host=127.0.0.1
--http-port=9652
--staking-port=9653
--db-dir=db/node2
--network-id=5678
--staking-tls-cert-file=$(pwd)/staking/local/staker2.crt
--staking-tls-key-file=$(pwd)/staking/local/staker2.key

./build/axia
--genesis=./genesis/genesis.json
--http-host=127.0.0.1
--http-port=9654
--staking-port=9655
--db-dir=db/node3
--network-id=5678
--staking-tls-cert-file=$(pwd)/staking/local/staker3.crt
--staking-tls-key-file=$(pwd)/staking/local/staker3.key

./build/axia
--genesis=./genesis/genesis.json
--http-host=127.0.0.1
--http-port=9656
--staking-port=9657
--db-dir=db/node4
--network-id=5678
--staking-tls-cert-file=$(pwd)/staking/local/staker4.crt
--staking-tls-key-file=$(pwd)/staking/local/staker4.key
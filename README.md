# P2P-Sync

To build peer to peer based utility to transfer large sized files from a single host to multiple hosts. Currently available utilities like rsync, scp can sync a file to one receiver at a time. So, for syncing a file to a large number of hosts, rsync/scp needs to be forked multiple times. This approach is time consuming and doesn't scale well because the sender has limited bandwidth. We want to leverage the power of P2P technology, so that all hosts which are involved in this process (one sender and multiple receivers) can send chunks of files to each other. In this way, bandwidth of all hosts will be utilized and every host will get the file in less time.

Our design is inspired from BitTorrent which is a popular P2P file transfer protocol.  Unlike BitTorrent, where peers which want to download the file initiate the process (pull based), our utility is push based. So nodes don’t ask for the file; whoever has the file, pushes it to other nodes

Main Components -
1. Master node, Destination nodes
2. p2p sync daemon running on all nodes
3. psync utility to trigger the sync
4. PeerInfoManager to track states of all the participating nodes
5. Chunk Creation
6. Chunk Selection
7. Chunk Aggregation
8. Communication Protocol
9. Bandwidth restrictor

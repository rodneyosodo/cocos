# Manager for Cocos AI

## Setup

```sh
git clone https://github.com/ultravioletrs/manager
cd manager
```

NB: all relative paths in this document are relative to `manager` repository directory.

### QEMU-KVM

[QEMU-KVM](https://www.qemu.org/) is a virtualization platform that allows you to run multiple operating systems on the same physical machine. It is a combination of two technologies: QEMU and KVM.

- QEMU is an emulator that can run a variety of operating systems, including Linux, Windows, and macOS.
- [KVM](https://wiki.qemu.org/Features/KVM) is a Linux kernel module that allows QEMU to run virtual machines.

To install QEMU-KVM on a Debian based machine, run

```sh
sudo apt update
sudo apt install qemu-kvm
```

Create `img` directory in `cmd/manager`.

### focal-server-cloudimg-amd64.img

First, we will download *focal-server-cloudimg-amd64*. It is a `qcow2` file with Ubuntu server preinstalled, ready to use with the QEMU virtual machine.

```sh
cd cmd/manager/img
wget https://cloud-images.ubuntu.com/focal/current/focal-server-cloudimg-amd64.img
# focal-server-cloudimg-amd64 comes without the root password.
sudo apt-get install libguestfs-tools
sudo virt-customize -a focal-server-cloudimg-amd64.img --root-password password:coolpass
```

### OVMF

We need [Open Virtual Machine Firmware](https://wiki.ubuntu.com/UEFI/OVMF). OVMF is a port of Intel's tianocore firmware - an open source implementation of the Unified Extensible Firmware Interface (UEFI) - to the qemu virtual machine. We need OVMF in order to run virtual machine with *focal-server-cloudimg-amd64*. When we install QEMU, we get two files that we need to start a VM: `OVMF_VARS.fd` and `OVMF_CODE.fd`. We will make a local copy of `OVMF_VARS.fd` since a VM will modify this file. On the other hand, `OVMF_CODE.fd` is only used as a reference, so we only record its path in an environment variable.

```sh
cd cmd/manager/img

sudo find / -name OVMF_CODE.fd
# => /usr/share/OVMF/OVMF_CODE.fd
# note this value

sudo find / -name OVMF_VARS.fd
# => /usr/share/OVMF/OVMF_VARS.fd
cp /usr/share/OVMF/OVMF_VARS.fd .
```

## Run manager

We need to run `manager` in the directory containing `img` directory:

```sh
cd cmd/manager
MANAGER_LOG_LEVEL=info MANAGER_AGENT_GRPC_URL=192.168.122.251:7002 go run main.go
```
Manager will start an HTTP server on port `9021`, and a gRPC server on port `7001`.

## Create QEMU virtual machine (VM)

### HTTP

```sh
cd cmd/manager
```

```sh
curl -sSi -X POST -H "Content-Type: application/json" http://localhost:9021/qemu -d @- <<EOF
{
  "config": {
    "use_sudo": false,
    "enable_sev": false,
    "enable_kvm": true,
    "machine": "q35",
    "cpu": "EPYC",
    "smp_count": 4,
    "smp_maxcpus": 64,
    "memory_size": "4096M",
    "memory_slots": 5,
    "max_memory": "30G",
    "ovmf_code_if": "pflash",
    "ovmf_code_format": "raw",
    "ovmf_code_unit": 0,
    "ovmf_code_file": "/usr/share/OVMF/OVMF_CODE.fd",
    "ovmf_code_readonly": "on",
    "ovmf_vars_if": "pflash",
    "ovmf_vars_format": "raw",
    "ovmf_vars_unit": 1,
    "ovmf_vars_file": "img/OVMF_VARS.fd",
    "netdev_id": "vmnic",
    "host_fwd_1": "2222",
    "guest_fwd_1": "22",
    "host_fwd_2": "9301",
    "guest_fwd_2": "9031",
    "host_fwd_3": "7020",
    "guest_fwd_3": "7002",
    "virtio_net_pci_disable_legacy": "on",
    "virtio_net_pci_iommu_platform": true,
    "virtio_net_pci_romfile": "",
    "disk_img_file": "img/focal-server-cloudimg-amd64.img",
    "disk_img_if": "none",
    "disk_img_id": "disk0",
    "disk_img_format": "qcow2",
    "virtio_scsi_pci_id": "scsi",
    "virtio_scsi_pci_disable_legacy": "on",
    "virtio_scsi_pci_iommu_platform": true,
    "sev_id": "sev0",
    "sev_cbitpos": 51,
    "sev_reduced_phys_bits": 1,
    "memory_encryption_sev0": "sev0",
    "no_graphic": true,
    "monitor": "pty"
  }
}
EOF
```

To enable [AMD SEV](https://www.amd.com/en/developer/sev.html) feature you need to use appropriate request body keys:

```sh
    "use_sudo": true,
    "enable_sev": true,
```

### Verifying VM launch

NB: To verify that the manager successfully launched the VM, you need to open two terminals on the same machine. In one terminal, you need to launch `go run main.go` (with the environment variables of choice) and in the other, you can run the verification commands.

To verify that the manager launched the VM successfully, run the following command:

```sh
ps aux | grep qemu-system-x86_64
```

You should get something similar to this
```
darko     324763 95.3  6.0 6398136 981044 ?      Sl   16:17   0:15 /usr/bin/qemu-system-x86_64 -enable-kvm -machine q35 -cpu EPYC -smp 4,maxcpus=64 -m 4096M,slots=5,maxmem=30G -drive if=pflash,format=raw,unit=0,file=/usr/share/OVMF/OVMF_CODE.fd,readonly=on -drive if=pflash,format=raw,unit=1,file=img/OVMF_VARS.fd -device virtio-scsi-pci,id=scsi,disable-legacy=on,iommu_platform=true -drive file=img/focal-server-cloudimg-amd64.img,if=none,id=disk0,format=qcow2 -device scsi-hd,drive=disk0 -netdev user,id=vmnic,hostfwd=tcp::2222-:22,hostfwd=tcp::9301-:9031,hostfwd=tcp::7020-:7002 -device virtio-net-pci,disable-legacy=on,iommu_platform=true,netdev=vmnic,romfile= -nographic -monitor pty
```

If you run a command as `sudo`, you should get the output similar to this one

```
root       37982  0.0  0.0   9444  4572 pts/0    S+   16:18   0:00 sudo /usr/local/bin/qemu-system-x86_64 -enable-kvm -machine q35 -cpu EPYC -smp 4,maxcpus=64 -m 4096M,slots=5,maxmem=30G -drive if=pflash,format=raw,unit=0,file=/usr/share/OVMF/OVMF_CODE.fd,readonly=on -drive if=pflash,format=raw,unit=1,file=img/OVMF_VARS.fd -device virtio-scsi-pci,id=scsi,disable-legacy=on,iommu_platform=true -drive file=img/focal-server-cloudimg-amd64.img,if=none,id=disk0,format=qcow2 -device scsi-hd,drive=disk0 -netdev user,id=vmnic,hostfwd=tcp::2222-:22,hostfwd=tcp::9301-:9031,hostfwd=tcp::7020-:7002 -device virtio-net-pci,disable-legacy=on,iommu_platform=true,netdev=vmnic,romfile= -object sev-guest,id=sev0,cbitpos=51,reduced-phys-bits=1 -machine memory-encryption=sev0 -nographic -monitor pty
root       37989  122 13.1 5345816 4252312 pts/0 Sl+  16:19   0:04 /usr/local/bin/qemu-system-x86_64 -enable-kvm -machine q35 -cpu EPYC -smp 4,maxcpus=64 -m 4096M,slots=5,maxmem=30G -drive if=pflash,format=raw,unit=0,file=/usr/share/OVMF/OVMF_CODE.fd,readonly=on -drive if=pflash,format=raw,unit=1,file=img/OVMF_VARS.fd -device virtio-scsi-pci,id=scsi,disable-legacy=on,iommu_platform=true -drive file=img/focal-server-cloudimg-amd64.img,if=none,id=disk0,format=qcow2 -device scsi-hd,drive=disk0 -netdev user,id=vmnic,hostfwd=tcp::2222-:22,hostfwd=tcp::9301-:9031,hostfwd=tcp::7020-:7002 -device virtio-net-pci,disable-legacy=on,iommu_platform=true,netdev=vmnic,romfile= -object sev-guest,id=sev0,cbitpos=51,reduced-phys-bits=1 -machine memory-encryption=sev0 -nographic -monitor pty
```

The two processes are due to the fact that we run the command `/usr/bin/qemu-system-x86_64` as `sudo`, so there is one process for `sudo` command and the other for `/usr/bin/qemu-system-x86_64`.

### Troubleshooting VM launch

If the `ps aux | grep qemu-system-x86_64` give you something like this

```
darko      13913  0.0  0.0      0     0 pts/2    Z+   20:17   0:00 [qemu-system-x86] <defunct>
```

means that the a QEMU virtual machine that is currently defunct, meaning that it is no longer running. More precisely, the defunct process in the output is also known as a ["zombie" process](https://en.wikipedia.org/wiki/Zombie_process).

You can troubleshoot the VM launch procedure by running directly `qemu-system-x86_64` command. When you run `manager` with `MANAGER_LOG_LEVEL=info` env var set, it prints out the entire command used to launch a VM. The relevant part of the log might look like this

```
{"level":"info","message":"/usr/bin/qemu-system-x86_64 -enable-kvm -machine q35 -cpu EPYC -smp 4,maxcpus=64 -m 4096M,slots=5,maxmem=30G -drive if=pflash,format=raw,unit=0,file=/usr/share/OVMF/OVMF_CODE.fd,readonly=on -drive if=pflash,format=raw,unit=1,file=img/OVMF_VARS.fd -device virtio-scsi-pci,id=scsi,disable-legacy=on,iommu_platform=true -drive file=img/focal-server-cloudimg-amd64.img,if=none,id=disk0,format=qcow2 -device scsi-hd,drive=disk0 -netdev user,id=vmnic,hostfwd=tcp::2222-:22,hostfwd=tcp::9301-:9031,hostfwd=tcp::7020-:7002 -device virtio-net-pci,disable-legacy=on,iommu_platform=true,netdev=vmnic,romfile= -nographic -monitor pty","ts":"2023-08-14T18:29:19.2653908Z"}
```

You can run the command - the value of the `"message"` key - directly in the terminal:

```sh
/usr/bin/qemu-system-x86_64 -enable-kvm -machine q35 -cpu EPYC -smp 4,maxcpus=64 -m 4096M,slots=5,maxmem=30G -drive if=pflash,format=raw,unit=0,file=/usr/share/OVMF/OVMF_CODE.fd,readonly=on -drive if=pflash,format=raw,unit=1,file=img/OVMF_VARS.fd -device virtio-scsi-pci,id=scsi,disable-legacy=on,iommu_platform=true -drive file=img/focal-server-cloudimg-amd64.img,if=none,id=disk0,format=qcow2 -device scsi-hd,drive=disk0 -netdev user,id=vmnic,hostfwd=tcp::2222-:22,hostfwd=tcp::9301-:9031,hostfwd=tcp::7020-:7002 -device virtio-net-pci,disable-legacy=on,iommu_platform=true,netdev=vmnic,romfile= -nographic -monitor pty
```

and look for the possible problems. This problems can usually be solved by using the adequate request body keys value assignments. Look in the `manager/qemu/config.go` file to see the recognized json keys.

#### Kill `qemu-system-x86_64` processes

To kill any leftover `qemu-system-x86_64` processes, use

```sh
pkill -f qemu-system-x86_64
```

The pkill command is used to kill processes by name or by pattern. The -f flag to specify that we want to kill processes that match the pattern `qemu-system-x86_64`. It sends the SIGKILL signal to all processes that are running `qemu-system-x86_64`.

If this does not work, i.e. if `ps aux | grep qemu-system-x86_64` still outputs `qemu-system-x86_64` related process(es), you can kill the unwanted process with `kill -9 <PID>`, which also sends a SIGKILL signal to the process.

#### Ports in use

The [NetDevConfig struct](manager/qemu/config.go) defines the network configuration for a virtual machine. The HostFwd* and GuestFwd* fields specify the host and guest ports that are forwarded between the virtual machine and the host machine. By default, these ports are allocated 2222, 9301, and 7020 for HostFwd1, HostFwd2, and HostFwd3, respectively, and 22, 9031, and 7002 for GuestFwd1, GuestFwd2, and GuestFwd3, respectively. However, if these ports are in use, you can configure your own ports by setting the corresponding body request keys. For example,

```sh
    "host_fwd_1": "2222",
    "guest_fwd_1": "22",
    "host_fwd_2": "9301",
    "guest_fwd_2": "9031",
    "host_fwd_3": "7020",
    "guest_fwd_3": "7002",
```
package main

import (
	"fmt"
	"github.com/lengrongfu/overlayfs-quota/quota-injector/quota"
)

func main() {
	qctr, err := quota.NewControl("/data")
	if err != nil {
		panic(err)
	}
	//q := quota.Quota{
	//	Size: 1024 * 1024 * 1024,
	//}
	//err = qctr.SetQuotaByProject("/data/run/containerd/io.containerd.runtime.v2.task/k8s.io/9cbc5cd12a19ee614640c28d88d1358673448ce13b54a309a89bd01c621f4f31/rootfs", q, 3)
	//if err != nil {
	//	panic(err)
	//}
	clears := []uint32{2, 3, 4}
	for _, p := range clears {
		err = qctr.ClearQuota(p)
		if err != nil {
			panic(err)
		}
	}
	fmt.Println("quota set success")
}

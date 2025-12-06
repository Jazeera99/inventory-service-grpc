package main

import (
	"context"
	"log"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	pb "inventory-service/proto" // sesuaikan dengan path kamu
)

func main() {
	// Koneksi ke server
	target := "135.79.1.12:50051"

	conn, err := grpc.Dial(target,
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("❌ Gagal koneksi: %v", err)
	}
	defer conn.Close()

	client := pb.NewInventoryServiceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	log.Println("=" * 60)
	log.Println("📦 SISTEM INVENTORY GUDANG")
	log.Println("=" * 60)

	// ========== SKENARIO 1: Tambah Item Baru ==========
	log.Println("\n🆕 SKENARIO 1: Tambah Item Baru")
	log.Println("-" * 40)

	newItem := &pb.AddItemRequest{
		Barcode:    "8996001111017",
		NamaBarang: "Kopi Kapal Api",
		QtyOnhand:  100,
	}

	addRes, err := client.AddItem(ctx, newItem)
	if err != nil {
		log.Printf("❌ Error: %v", err)
	} else {
		log.Printf("✅ %s", addRes.Message)
		log.Printf("   [%s] %s - Stok Awal: %d",
			addRes.Item.Barcode, addRes.Item.NamaBarang, addRes.Item.QtyOnhand)
	}

	// ========== SKENARIO 2: Scan Barcode ==========
	log.Println("\n📱 SKENARIO 2: Scan Barcode")
	log.Println("-" * 40)

	barcodes := []string{"8992761111014", "8992388888014", "8991002111028"}

	for _, barcode := range barcodes {
		item, err := client.ScanBarcode(ctx, &pb.ScanBarcodeRequest{Barcode: barcode})
		if err != nil {
			log.Printf("❌ Scan gagal: %v", err)
			continue
		}
		log.Printf("   [%s] %s - Stok: %d unit",
			item.Barcode, item.NamaBarang, item.QtyOnhand)
	}

	// ========== SKENARIO 3: Barang Masuk (Pembelian) ==========
	log.Println("\n📦 SKENARIO 3: Barang Masuk - Pembelian")
	log.Println("-" * 40)

	masukRes, err := client.UpdateStock(ctx, &pb.UpdateStockRequest{
		Barcode:    "8992761111014",
		Quantity:   50,
		Keterangan: "Pembelian dari supplier",
	})
	if err != nil {
		log.Printf("❌ Error: %v", err)
	} else {
		log.Printf("✅ %s", masukRes.Message)
		log.Printf("   %s: %d → %d (+%d)",
			masukRes.Item.NamaBarang,
			masukRes.StokSebelum,
			masukRes.StokSesudah,
			masukRes.StokSesudah-masukRes.StokSebelum)
	}

	// ========== SKENARIO 4: Barang Keluar (Penjualan) ==========
	log.Println("\n📤 SKENARIO 4: Barang Keluar - Penjualan")
	log.Println("-" * 40)

	keluarRes, err := client.UpdateStock(ctx, &pb.UpdateStockRequest{
		Barcode:    "8992388888014",
		Quantity:   -30, // Negatif = barang keluar
		Keterangan: "Penjualan kasir 1",
	})
	if err != nil {
		log.Printf("❌ Error: %v", err)
	} else {
		log.Printf("✅ %s", keluarRes.Message)
		log.Printf("   %s: %d → %d (%d)",
			keluarRes.Item.NamaBarang,
			keluarRes.StokSebelum,
			keluarRes.StokSesudah,
			keluarRes.StokSesudah-keluarRes.StokSebelum)
	}

	// ========== SKENARIO 5: Cek Stok ==========
	log.Println("\n🔍 SKENARIO 5: Cek Status Stok")
	log.Println("-" * 40)

	checkBarcodes := []string{"8992761111014", "8991002111028"}

	for _, barcode := range checkBarcodes {
		checkRes, err := client.CheckStock(ctx, &pb.CheckStockRequest{Barcode: barcode})
		if err != nil {
			log.Printf("❌ Error: %v", err)
			continue
		}
		log.Printf("   [%s] %s", checkRes.Barcode, checkRes.NamaBarang)
		log.Printf("   Stok: %d unit - Status: %s",
			checkRes.QtyOnhand, checkRes.Status)
	}

	// ========== SKENARIO 6: List Stok Rendah ==========
	log.Println("\n⚠️  SKENARIO 6: Alert - Stok Rendah")
	log.Println("-" * 40)

	lowStockRes, err := client.ListItems(ctx, &pb.ListItemsRequest{
		OnlyLowStock:      true,
		LowStockThreshold: 10,
	})
	if err != nil {
		log.Printf("❌ Error: %v", err)
	} else {
		if lowStockRes.Total == 0 {
			log.Println("   ✅ Semua stok aman!")
		} else {
			log.Printf("   🚨 Ditemukan %d item dengan stok rendah:", lowStockRes.Total)
			for _, item := range lowStockRes.Items {
				log.Printf("      - [%s] %s: %d unit",
					item.Barcode, item.NamaBarang, item.QtyOnhand)
			}
		}
	}

	// ========== SKENARIO 7: List Semua Item ==========
	log.Println("\n📊 SKENARIO 7: Laporan Inventory Lengkap")
	log.Println("-" * 40)

	allItemsRes, err := client.ListItems(ctx, &pb.ListItemsRequest{})
	if err != nil {
		log.Printf("❌ Error: %v", err)
	} else {
		log.Printf("   Total Item: %d", allItemsRes.Total)
		log.Println("\n   Barcode          | Nama Barang              | Qty")
		log.Println("   " + ("-" * 55))

		totalStok := int32(0)
		for _, item := range allItemsRes.Items {
			log.Printf("   %-16s | %-24s | %d",
				item.Barcode, item.NamaBarang, item.QtyOnhand)
			totalStok += item.QtyOnhand
		}
		log.Println("   " + ("-" * 55))
		log.Printf("   Total Stok: %d unit\n", totalStok)
	}

	log.Println("=" * 60)
}

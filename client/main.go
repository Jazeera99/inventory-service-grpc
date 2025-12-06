package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	// Import package proto
	pb "inventory-service/proto"
)

func main() {
	// 1. Koneksi ke Server
	// KITA PAKAI LOCALHOST.
	// Saat server di Debian pun, karena pakai Port Forwarding, Windows tetap nembak ke localhost.
	target := "localhost:50051"

	conn, err := grpc.Dial(target, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("❌ Gagal koneksi ke server: %v", err)
	}
	defer conn.Close()

	client := pb.NewInventoryServiceClient(conn)

	// Context dengan timeout agar client tidak hang selamanya jika server mati
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	fmt.Println("========================================")
	fmt.Println("       SISTEM INVENTORY GUDANG (gRPC)   ")
	fmt.Println("========================================")

	// --- SKENARIO 1: Tambah Item Baru ---
	printHeader("1. Tambah Item Baru")
	newItem := &pb.AddItemRequest{
		Barcode:    "8996001111017",
		NamaBarang: "Kopi Kapal Api",
		QtyOnhand:  100,
	}
	addRes, err := client.AddItem(ctx, newItem)
	if err != nil {
		log.Printf("❌ Error: %v", err)
	} else {
		log.Printf("✅ Sukses: %s", addRes.Message)
		log.Printf("   Data: [%s] %s | Stok Awal: %d", addRes.Item.Barcode, addRes.Item.NamaBarang, addRes.Item.QtyOnhand)
	}

	// --- SKENARIO 2: Scan Barcode ---
	printHeader("2. Scan Barcode")
	barcodes := []string{"8992761111014", "8992388888014", "8991002111028"}
	for _, b := range barcodes {
		item, err := client.ScanBarcode(ctx, &pb.ScanBarcodeRequest{Barcode: b})
		if err != nil {
			log.Printf("❌ Scan Gagal (%s): %v", b, err)
		} else {
			log.Printf("🔎 Ditemukan: [%s] %s | Stok: %d", item.Barcode, item.NamaBarang, item.QtyOnhand)
		}
	}

	// --- SKENARIO 3: Barang Masuk (Pembelian) ---
	printHeader("3. Barang Masuk (Restock)")
	masukRes, err := client.UpdateStock(ctx, &pb.UpdateStockRequest{
		Barcode:    "8992761111014", // Indomie
		Quantity:   50,
		Keterangan: "Pembelian dari Supplier",
	})
	if err != nil {
		log.Printf("❌ Error: %v", err)
	} else {
		log.Printf("✅ %s", masukRes.Message)
		log.Printf("   Update: %s (%d -> %d)", masukRes.Item.NamaBarang, masukRes.StokSebelum, masukRes.StokSesudah)
	}

	// --- SKENARIO 4: Barang Keluar (Penjualan) ---
	printHeader("4. Barang Keluar (Penjualan)")
	keluarRes, err := client.UpdateStock(ctx, &pb.UpdateStockRequest{
		Barcode:    "8992388888014", // Aqua
		Quantity:   -30,             // Negatif = Keluar
		Keterangan: "Penjualan Kasir 1",
	})
	if err != nil {
		log.Printf("❌ Error: %v", err)
	} else {
		log.Printf("✅ %s", keluarRes.Message)
		log.Printf("   Update: %s (%d -> %d)", keluarRes.Item.NamaBarang, keluarRes.StokSebelum, keluarRes.StokSesudah)
	}

	// --- SKENARIO 5: Cek Status Stok ---
	printHeader("5. Cek Status Stok")
	checkRes, err := client.CheckStock(ctx, &pb.CheckStockRequest{Barcode: "8991002111028"}) // Teh Botol (Stok dikit)
	if err != nil {
		log.Printf("❌ Error: %v", err)
	} else {
		log.Printf("📊 Status: [%s] %s", checkRes.Barcode, checkRes.NamaBarang)
		log.Printf("   Stok: %d | Status: %s", checkRes.QtyOnhand, checkRes.Status)
	}

	// --- SKENARIO 6: Reporting (List All) ---
	printHeader("6. Laporan Inventory Lengkap")
	listRes, err := client.ListItems(ctx, &pb.ListItemsRequest{})
	if err != nil {
		log.Printf("❌ Error: %v", err)
	} else {
		log.Printf("Total Item: %d", listRes.Total)
		fmt.Println("--------------------------------------------------")
		fmt.Printf("%-15s | %-20s | %s\n", "Barcode", "Nama Barang", "Qty")
		fmt.Println("--------------------------------------------------")
		for _, item := range listRes.Items {
			fmt.Printf("%-15s | %-20s | %d\n", item.Barcode, item.NamaBarang, item.QtyOnhand)
		}
		fmt.Println("--------------------------------------------------")
	}
}

func printHeader(title string) {
	fmt.Println("\n----------------------------------------")
	fmt.Printf(">> %s\n", title)
	fmt.Println("----------------------------------------")
	time.Sleep(500 * time.Millisecond) // Biar output gak balapan
}

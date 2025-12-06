package main

import (
	"context"
	"fmt"
	"log"
	"net"

	pb "path/to/your/proto" // sesuaikan dengan path kamu

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type inventoryServer struct {
	pb.UnimplementedInventoryServiceServer
	items map[string]*pb.Item // key: barcode, value: Item
}

func newServer() *inventoryServer {
	s := &inventoryServer{
		items: make(map[string]*pb.Item),
	}

	// Data dummy untuk testing
	s.items["8992761111014"] = &pb.Item{
		Barcode:    "8992761111014",
		NamaBarang: "Indomie Goreng",
		QtyOnhand:  150,
	}
	s.items["8992388888014"] = &pb.Item{
		Barcode:    "8992388888014",
		NamaBarang: "Aqua 600ml",
		QtyOnhand:  200,
	}
	s.items["8991002111028"] = &pb.Item{
		Barcode:    "8991002111028",
		NamaBarang: "Teh Botol Sosro",
		QtyOnhand:  5, // Stok rendah
	}
	return s
}

// Tambah item baru ke inventory
func (s *inventoryServer) AddItem(ctx context.Context, req *pb.AddItemRequest) (*pb.AddItemResponse, error) {
	// Validasi barcode tidak boleh kosong
	if req.Barcode == "" {
		return nil, status.Error(codes.InvalidArgument, "barcode tidak boleh kosong")
	}

	// Cek apakah barcode sudah ada
	if _, exists := s.items[req.Barcode]; exists {
		return nil, status.Error(codes.AlreadyExists, "barcode sudah terdaftar")
	}

	// Buat item baru
	item := &pb.Item{
		Barcode:    req.Barcode,
		NamaBarang: req.NamaBarang,
		QtyOnhand:  req.QtyOnhand,
	}

	s.items[req.Barcode] = item

	log.Printf("✅ Item baru ditambahkan: [%s] %s (Stok: %d)",
		item.Barcode, item.NamaBarang, item.QtyOnhand)

	return &pb.AddItemResponse{
		Item:    item,
		Message: "Item berhasil ditambahkan ke inventory",
	}, nil
}

// Scan barcode untuk mendapatkan info item
func (s *inventoryServer) ScanBarcode(ctx context.Context, req *pb.ScanBarcodeRequest) (*pb.Item, error) {
	item, exists := s.items[req.Barcode]
	if !exists {
		return nil, status.Error(codes.NotFound,
			fmt.Sprintf("barcode %s tidak ditemukan di sistem", req.Barcode))
	}

	log.Printf("📱 Scan: [%s] %s - Stok: %d",
		item.Barcode, item.NamaBarang, item.QtyOnhand)

	return item, nil
}

// Update stok (barang masuk atau keluar)
func (s *inventoryServer) UpdateStock(ctx context.Context, req *pb.UpdateStockRequest) (*pb.UpdateStockResponse, error) {
	item, exists := s.items[req.Barcode]
	if !exists {
		return nil, status.Error(codes.NotFound, "barcode tidak ditemukan")
	}

	stokSebelum := item.QtyOnhand
	stokSesudah := stokSebelum + req.Quantity

	// Validasi: stok tidak boleh negatif
	if stokSesudah < 0 {
		return nil, status.Error(codes.InvalidArgument,
			fmt.Sprintf("stok tidak cukup. Tersedia: %d, Diminta: %d",
				stokSebelum, -req.Quantity))
	}

	// Update stok
	item.QtyOnhand = stokSesudah

	// Log activity
	if req.Quantity > 0 {
		log.Printf("📦 MASUK: [%s] %s +%d (Stok: %d → %d) - %s",
			item.Barcode, item.NamaBarang, req.Quantity,
			stokSebelum, stokSesudah, req.Keterangan)
	} else {
		log.Printf("📤 KELUAR: [%s] %s %d (Stok: %d → %d) - %s",
			item.Barcode, item.NamaBarang, req.Quantity,
			stokSebelum, stokSesudah, req.Keterangan)
	}

	return &pb.UpdateStockResponse{
		Item:        item,
		Message:     "Stok berhasil diupdate",
		StokSebelum: stokSebelum,
		StokSesudah: stokSesudah,
	}, nil
}

// Cek stok dan status ketersediaan
func (s *inventoryServer) CheckStock(ctx context.Context, req *pb.CheckStockRequest) (*pb.CheckStockResponse, error) {
	item, exists := s.items[req.Barcode]
	if !exists {
		return nil, status.Error(codes.NotFound, "barcode tidak ditemukan")
	}

	// Tentukan status berdasarkan stok
	var status string
	switch {
	case item.QtyOnhand == 0:
		status = "❌ HABIS"
	case item.QtyOnhand <= 10:
		status = "⚠️  STOK RENDAH"
	default:
		status = "✅ TERSEDIA"
	}

	log.Printf("🔍 Cek Stok: [%s] %s = %d unit (%s)",
		item.Barcode, item.NamaBarang, item.QtyOnhand, status)

	return &pb.CheckStockResponse{
		Barcode:    item.Barcode,
		NamaBarang: item.NamaBarang,
		QtyOnhand:  item.QtyOnhand,
		Status:     status,
	}, nil
}

// List semua item atau filter stok rendah
func (s *inventoryServer) ListItems(ctx context.Context, req *pb.ListItemsRequest) (*pb.ListItemsResponse, error) {
	var items []*pb.Item
	threshold := req.LowStockThreshold
	if threshold == 0 {
		threshold = 10 // Default threshold
	}

	for _, item := range s.items {
		// Filter jika diminta hanya low stock
		if req.OnlyLowStock && item.QtyOnhand > threshold {
			continue
		}
		items = append(items, item)
	}

	if req.OnlyLowStock {
		log.Printf("📊 Menampilkan %d item dengan stok <= %d", len(items), threshold)
	} else {
		log.Printf("📊 Menampilkan semua %d item", len(items))
	}

	return &pb.ListItemsResponse{
		Items: items,
		Total: int32(len(items)),
	}, nil
}

func main() {
	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("❌ Gagal listen: %v", err)
	}

	grpcServer := grpc.NewServer()
	pb.RegisterInventoryServiceServer(grpcServer, newServer())

	log.Println("🚀 Inventory Server berjalan di port 50051...")
	log.Println("📦 Siap menerima request dari barcode scanner!")

	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("❌ Gagal serve: %v", err)
	}
}

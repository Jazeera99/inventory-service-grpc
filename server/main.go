package main

import (
	"context"
	"log"
	"net"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	// Import package proto yang baru saja kita generate
	// Pastikan nama module di go.mod adalah "inventory-service"
	pb "inventory-service/proto"
)

// inventoryServer adalah struct utama server kita
type inventoryServer struct {
	pb.UnimplementedInventoryServiceServer
	items map[string]*pb.Item // Database sederhana menggunakan Map (RAM)
}

// newServer menginisialisasi server dengan data dummy
func newServer() *inventoryServer {
	s := &inventoryServer{
		items: make(map[string]*pb.Item),
	}

	// Data dummy sesuai PDF [cite: 115-127]
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
		QtyOnhand:  5, // Stok rendah untuk simulasi
	}

	return s
}

// 1. AddItem: Menambah barang baru
func (s *inventoryServer) AddItem(ctx context.Context, req *pb.AddItemRequest) (*pb.AddItemResponse, error) {
	if req.Barcode == "" {
		return nil, status.Error(codes.InvalidArgument, "barcode tidak boleh kosong")
	}

	if _, exists := s.items[req.Barcode]; exists {
		return nil, status.Error(codes.AlreadyExists, "barcode sudah terdaftar")
	}

	item := &pb.Item{
		Barcode:    req.Barcode,
		NamaBarang: req.NamaBarang,
		QtyOnhand:  req.QtyOnhand,
	}

	s.items[req.Barcode] = item
	log.Printf("📥 Item Baru: [%s] %s (Stok: %d)", item.Barcode, item.NamaBarang, item.QtyOnhand)

	return &pb.AddItemResponse{
		Item:    item,
		Message: "Item berhasil ditambahkan ke inventory",
	}, nil
}

// 2. ScanBarcode: Mencari barang by barcode
func (s *inventoryServer) ScanBarcode(ctx context.Context, req *pb.ScanBarcodeRequest) (*pb.Item, error) {
	item, exists := s.items[req.Barcode]
	if !exists {
		return nil, status.Errorf(codes.NotFound, "barcode %s tidak ditemukan", req.Barcode)
	}
	log.Printf("🔍 Scan: [%s] %s Stok: %d", item.Barcode, item.NamaBarang, item.QtyOnhand)
	return item, nil
}

// 3. UpdateStock: Barang Masuk/Keluar
func (s *inventoryServer) UpdateStock(ctx context.Context, req *pb.UpdateStockRequest) (*pb.UpdateStockResponse, error) {
	item, exists := s.items[req.Barcode]
	if !exists {
		return nil, status.Error(codes.NotFound, "barcode tidak ditemukan")
	}

	stokSebelum := item.QtyOnhand
	stokSesudah := stokSebelum + req.Quantity

	if stokSesudah < 0 {
		return nil, status.Errorf(codes.InvalidArgument, "stok tidak cukup. Tersedia: %d, Diminta Keluar: %d", stokSebelum, -req.Quantity)
	}

	item.QtyOnhand = stokSesudah

	// Log activity
	action := "KELUAR"
	if req.Quantity > 0 {
		action = "MASUK"
	}
	log.Printf("📦 UPDATE %s: [%s] %s %d (Stok: %d -> %d) | Note: %s",
		action, item.Barcode, item.NamaBarang, req.Quantity, stokSebelum, stokSesudah, req.Keterangan)

	return &pb.UpdateStockResponse{
		Item:        item,
		Message:     "Stok berhasil diupdate",
		StokSebelum: stokSebelum,
		StokSesudah: stokSesudah,
	}, nil
}

// 4. CheckStock: Cek status stok
func (s *inventoryServer) CheckStock(ctx context.Context, req *pb.CheckStockRequest) (*pb.CheckStockResponse, error) {
	item, exists := s.items[req.Barcode]
	if !exists {
		return nil, status.Error(codes.NotFound, "barcode tidak ditemukan")
	}

	statusStr := "TERSEDIA"
	if item.QtyOnhand == 0 {
		statusStr = "HABIS"
	} else if item.QtyOnhand <= 10 {
		statusStr = "STOK RENDAH"
	}

	return &pb.CheckStockResponse{
		Barcode:    item.Barcode,
		NamaBarang: item.NamaBarang,
		QtyOnhand:  item.QtyOnhand,
		Status:     statusStr,
	}, nil
}

// 5. ListItems: Menampilkan semua item
func (s *inventoryServer) ListItems(ctx context.Context, req *pb.ListItemsRequest) (*pb.ListItemsResponse, error) {
	var items []*pb.Item
	threshold := req.LowStockThreshold
	if threshold == 0 {
		threshold = 10
	}

	for _, item := range s.items {
		if req.OnlyLowStock && item.QtyOnhand > threshold {
			continue
		}
		items = append(items, item)
	}

	return &pb.ListItemsResponse{
		Items: items,
		Total: int32(len(items)),
	}, nil
}

func main() {
	// Listen di port 50051 (Port Standar gRPC)
	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("❌ Gagal listen: %v", err)
	}

	grpcServer := grpc.NewServer()
	pb.RegisterInventoryServiceServer(grpcServer, newServer())

	log.Println("✅ Inventory Server berjalan di port 50051...")
	log.Println("🚀 Siap menerima request dari Client...")

	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("❌ Gagal serve: %v", err)
	}
}

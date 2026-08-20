package actions

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/smtp"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gobuffalo/buffalo"
	"github.com/gobuffalo/nulls"
	"github.com/gobuffalo/pop/v6"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"backend_server/models"
	"backend_server/storage"
)

type RegistrantsResource struct {
	buffalo.Resource
	storageService *storage.StorageService
}

func NewRegistrantsResource() RegistrantsResource {
	return RegistrantsResource{
		storageService: storage.NewStorageService(),
	}
}

func (v RegistrantsResource) List(c buffalo.Context) error {
	tx, ok := c.Value("tx").(*pop.Connection)
	if !ok {
		return Response(c, http.StatusInternalServerError, "Database error", nil)
	}

	registrants := &models.Registrants{}
	q := tx.PaginateFromParams(c.Params())
	if err := q.All(registrants); err != nil {
		return Response(c, http.StatusInternalServerError, "Failed to retrieve data", err.Error())
	}

	for i := range *registrants {
		(*registrants)[i].Password = nulls.NewString("")
	}

	return Response(c, http.StatusOK, "Success", map[string]interface{}{
		"data":       registrants,
		"pagination": q.Paginator,
	})
}

func (v RegistrantsResource) Show(c buffalo.Context) error {
	tx, ok := c.Value("tx").(*pop.Connection)
	if !ok {
		return Response(c, http.StatusInternalServerError, "Database error", nil)
	}

	registrant := &models.Registrant{}
	if err := tx.Find(registrant, c.Param("registrant_id")); err != nil {
		return Response(c, http.StatusNotFound, "Registrant not found", nil)
	}

	registrant.Password = nulls.NewString("")
	return Response(c, http.StatusOK, "Success", registrant)
}

// FUNGSI INI DIMATIKAN SECARA PERMANEN
func (v RegistrantsResource) Create(c buffalo.Context) error {
	return Response(c, http.StatusMethodNotAllowed, "Endpoint dinonaktifkan. Pendaftaran sekarang wajib melalui SSO Google.", nil)
}

func (v RegistrantsResource) Destroy(c buffalo.Context) error {
	tx, ok := c.Value("tx").(*pop.Connection)
	if !ok {
		return Response(c, http.StatusInternalServerError, "Database error", nil)
	}

	registrant := &models.Registrant{}
	if err := tx.Find(registrant, c.Param("registrant_id")); err != nil {
		return Response(c, http.StatusNotFound, "Registrant not found", nil)
	}

	if registrant.CvUrl.Valid && registrant.CvUrl.String != "" {
		key := v.storageService.ExtractObjectKey(registrant.CvUrl.String)
		if err := v.storageService.Delete(c.Request().Context(), key); err != nil {
			log.Printf("⚠️ gagal hapus CV lokal: %v", err)
		}
	}

	if err := tx.Destroy(registrant); err != nil {
		return Response(c, http.StatusInternalServerError, "Failed to delete", err.Error())
	}

	return Response(c, http.StatusOK, "Deleted successfully", nil)
}

func (v RegistrantsResource) Login(c buffalo.Context) error {
	// ... (kode bind JSON tetap sama seperti aslinya)
	var input struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := c.Bind(&input); err != nil {
		return Response(c, http.StatusBadRequest, "Invalid JSON body", nil)
	}

	tx, ok := c.Value("tx").(*pop.Connection)
	if !ok {
		return Response(c, http.StatusInternalServerError, "Database error", nil)
	}

	registrant := &models.Registrant{}
	if err := tx.Where("email = ?", input.Email).First(registrant); err != nil {
		return Response(c, http.StatusUnauthorized, "Email not found", nil)
	}

	if !registrant.Password.Valid || registrant.Password.String == "" {
		return Response(c, http.StatusForbidden, "Akun ini terdaftar menggunakan Google SSO. Silakan login via Google.", nil)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(registrant.Password.String), []byte(input.Password)); err != nil {
		return Response(c, http.StatusUnauthorized, "Wrong password", nil)
	}

	// PENTING: Role "registrant"
	token, err := JWTService.CreateJWTToken(registrant.ID, "registrant")
	if err != nil {
		return Response(c, http.StatusInternalServerError, "Failed to generate token", err.Error())
	}

	data := map[string]interface{}{
		"registrant_id": registrant.ID,
		"name":          registrant.Name,
		"email":         registrant.Email,
		"status":        registrant.Status,
		"token":         token,
	}
	return Response(c, http.StatusOK, "Login successful", data)
}

func (v RegistrantsResource) GoogleLogin(c buffalo.Context) error {
	// ... (kode ambil dan ekstrak JSON dari Google sama persis)
	var input struct {
		Token string `json:"google_token"`
	}
	if err := c.Bind(&input); err != nil {
		return Response(c, http.StatusBadRequest, "Invalid JSON payload", nil)
	}

	resp, err := http.Get("https://oauth2.googleapis.com/tokeninfo?id_token=" + input.Token)
	if err != nil || resp.StatusCode != http.StatusOK {
		return Response(c, http.StatusUnauthorized, "Invalid Google Token", nil)
	}
	defer resp.Body.Close()

	var googleData struct {
		Email string `json:"email"`
		Name  string `json:"name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&googleData); err != nil {
		return Response(c, http.StatusInternalServerError, "Failed to parse Google response", nil)
	}

	if !strings.HasSuffix(googleData.Email, "@student.ub.ac.id") {
		return Response(c, http.StatusForbidden, "Hanya email @student.ub.ac.id yang diizinkan mendaftar", nil)
	}

	tx, ok := c.Value("tx").(*pop.Connection)
	if !ok {
		return Response(c, http.StatusInternalServerError, "Database error", nil)
	}

	registrant := &models.Registrant{}
	err = tx.Where("email = ?", googleData.Email).First(registrant)
	
	if err != nil {
		if googleData.Name == "" {
			googleData.Name = strings.Split(googleData.Email, "@")[0]
		}

		registrant = &models.Registrant{
			Name:   googleData.Name,
			Email:  googleData.Email,
			Status: "PENDING",
		}
		
		verrs, err := tx.ValidateAndCreate(registrant)
		if err != nil {
			return Response(c, http.StatusInternalServerError, "Gagal membuat akun otomatis", err.Error())
		}
		if verrs.HasAny() {
			return Response(c, http.StatusUnprocessableEntity, "Validasi auto-register gagal", verrs)
		}
	}

	// PENTING: Role "registrant"
	token, err := JWTService.CreateJWTToken(registrant.ID, "registrant")
	if err != nil {
		return Response(c, http.StatusInternalServerError, "Failed to generate token", err.Error())
	}

	data := map[string]interface{}{
		"registrant_id": registrant.ID,
		"name":          registrant.Name,
		"email":         registrant.Email,
		"status":        registrant.Status,
		"token":         token,
	}
	return Response(c, http.StatusOK, "Login Google berhasil", data)
}

func (v RegistrantsResource) GetMe(c buffalo.Context) error {
	role, _ := c.Value("role").(string)
	if role != "registrant" {
		return Response(c, http.StatusForbidden, "Hanya Pendaftar yang bisa mengakses fitur ini", nil)
	}

	userIDStr, _ := c.Value("user_id").(string)
	tx, _ := c.Value("tx").(*pop.Connection)

	registrant := &models.Registrant{}
	if err := tx.Find(registrant, userIDStr); err != nil {
		return Response(c, http.StatusNotFound, "Registrant not found", nil)
	}

	registrant.Password = nulls.NewString("")
	return Response(c, http.StatusOK, "Success", registrant)
}

func (v RegistrantsResource) UpdateMe(c buffalo.Context) error {
	role, _ := c.Value("role").(string)
	if role != "registrant" {
		return Response(c, http.StatusForbidden, "Hanya Pendaftar yang bisa mengakses fitur ini", nil)
	}

	userIDStr, _ := c.Value("user_id").(string)
	tx, _ := c.Value("tx").(*pop.Connection)

	registrant := &models.Registrant{}
	if err := tx.Find(registrant, userIDStr); err != nil {
		return Response(c, http.StatusNotFound, "Registrant not found", nil)
	}

	// ... (Sisa kode Bind JSON & Update Field biarkan sama persis dengan fungsi UpdateMe aslimu)
	var input models.Registrant
	if err := c.Bind(&input); err != nil {
		return Response(c, http.StatusBadRequest, "Invalid JSON data", err.Error())
	}

	if input.CvUrl.Valid && input.CvUrl.String != "" && input.CvUrl.String != registrant.CvUrl.String {
		if registrant.CvUrl.Valid && registrant.CvUrl.String != "" {
			key := v.storageService.ExtractObjectKey(registrant.CvUrl.String)
			v.storageService.Delete(c.Request().Context(), key)
		}
		registrant.CvUrl = input.CvUrl
	}

	registrant.Name = input.Name
	registrant.NIM = input.NIM
	registrant.Angkatan = input.Angkatan
	registrant.Prodi = input.Prodi
	registrant.Fakultas = input.Fakultas
	registrant.Domicile = input.Domicile
	registrant.Phone = input.Phone
	registrant.Division1 = input.Division1
	registrant.Division2 = input.Division2
	registrant.SwotS = input.SwotS
	registrant.SwotW = input.SwotW
	registrant.SwotO = input.SwotO
	registrant.SwotT = input.SwotT
	registrant.OrganizationExp = input.OrganizationExp
	registrant.Commitment = input.Commitment

	verrs, err := tx.ValidateAndUpdate(registrant)
	if err != nil {
		return Response(c, http.StatusInternalServerError, "Failed to update profile", err.Error())
	}
	if verrs.HasAny() {
		return Response(c, http.StatusUnprocessableEntity, "Validation error", verrs)
	}

	registrant.Password = nulls.NewString("")
	return Response(c, http.StatusOK, "Profile updated successfully", registrant)
}

func (v RegistrantsResource) UploadCV(c buffalo.Context) error {
	role, _ := c.Value("role").(string)
	if role != "registrant" {
		return Response(c, http.StatusForbidden, "Hanya Pendaftar yang diizinkan mengunggah CV", nil)
	}

	// ... (Sisa kode cek ukuran, simpan file biarkan sama persis dengan aslimu)
	c.Request().Body = http.MaxBytesReader(c.Response(), c.Request().Body, 3<<20)

	file, err := c.File("cv")
	if err != nil {
		return Response(c, http.StatusBadRequest, "CV is required or file too large (Max 3MB)", err.Error())
	}
	defer file.Close()

	ext := strings.ToLower(filepath.Ext(file.Filename))
	if ext != ".pdf" {
		return Response(c, http.StatusBadRequest, "Only PDF files are allowed", nil)
	}

	safeFilename := strings.ReplaceAll(file.Filename, " ", "_")
	url, err := v.storageService.Upload(c.Request().Context(), "cv/"+uuid.New().String()+"-"+safeFilename, file)
	if err != nil {
		return Response(c, http.StatusInternalServerError, "Failed to save CV", err.Error())
	}

	return Response(c, http.StatusOK, "Upload success", map[string]string{
		"url": url,
	})
}
func (v RegistrantsResource) UpdateStatus(c buffalo.Context) error {
	var input struct {
		Status string `json:"status"`
	}
	if err := c.Bind(&input); err != nil {
		return Response(c, http.StatusBadRequest, "Invalid JSON body", nil)
	}

	tx, ok := c.Value("tx").(*pop.Connection)
	if !ok {
		return Response(c, http.StatusInternalServerError, "Database error", nil)
	}

	registrant := &models.Registrant{}
	if err := tx.Find(registrant, c.Param("registrant_id")); err != nil {
		return Response(c, http.StatusNotFound, "Registrant not found", nil)
	}

	registrant.Status = input.Status
	if err := tx.Update(registrant); err != nil {
		return Response(c, http.StatusInternalServerError, "Failed to update status", err.Error())
	}

	registrant.Password = nulls.NewString("")
	return Response(c, http.StatusOK, "Status updated successfully", registrant)
}

func (v RegistrantsResource) SendSchedule(c buffalo.Context) error {
	var input struct {
		RegistrantIDs     []string `json:"registrant_ids"`
		ScreeningDate     string   `json:"screening_date"` 
		ScreeningLocation string   `json:"screening_location"`
		ScreeningLink     string   `json:"screening_link"`
	}
	
	if err := c.Bind(&input); err != nil {
		return Response(c, http.StatusBadRequest, "Invalid JSON body", nil)
	}

	tx, ok := c.Value("tx").(*pop.Connection)
	if !ok {
		return Response(c, http.StatusInternalServerError, "Database error", nil)
	}

	parsedDate, _ := time.Parse("2006-01-02 15:04:05", input.ScreeningDate)
	var targets []models.Registrant

	for _, idStr := range input.RegistrantIDs {
		reg := models.Registrant{}
		if err := tx.Find(&reg, idStr); err == nil {
			reg.ScreeningDate = nulls.NewTime(parsedDate)
			reg.ScreeningLocation = nulls.NewString(input.ScreeningLocation)
			reg.ScreeningLink = nulls.NewString(input.ScreeningLink)
			tx.Update(&reg)
			
			targets = append(targets, reg)
		}
	}

	go sendScheduleEmailsAsync(targets, parsedDate.Format("02 Jan 2006 15:04 WIB"), input.ScreeningLocation, input.ScreeningLink)

	return Response(c, http.StatusOK, fmt.Sprintf("Proses pengiriman ke %d pendaftar sedang berjalan di latar belakang", len(targets)), nil)
}

func sendScheduleEmailsAsync(targets []models.Registrant, dateStr, location, link string) {
	senderEmail := os.Getenv("SMTP_EMAIL") 
	senderPassword := os.Getenv("SMTP_PASSWORD")
	smtpHost := "smtp.gmail.com"
	smtpPort := "587"

	if senderEmail == "" || senderPassword == "" {
		log.Println("⚠️ SMTP_EMAIL atau SMTP_PASSWORD kosong. Email dibatalkan.")
		return
	}

	auth := smtp.PlainAuth("", senderEmail, senderPassword, smtpHost)

	for _, user := range targets {
		subject := "Jadwal Screening Staff UKM UAKI\n"
		body := fmt.Sprintf("Halo %s,\n\nSelamat kamu lolos tahap berkas! Berikut adalah jadwal screening-mu:\nTanggal: %s\nLokasi: %s\nLink Meet: %s\n\nHarap hadir tepat waktu.\n\nSalam,\nPanitia OPREC UAKI", user.Name, dateStr, location, link)
		
		msg := []byte("To: " + user.Email + "\r\n" +
			"Subject: " + subject + "\r\n" +
			"\r\n" + body + "\r\n")

		err := smtp.SendMail(smtpHost+":"+smtpPort, auth, senderEmail, []string{user.Email}, msg)
		if err != nil {
			log.Printf("❌ Gagal kirim email ke %s: %v\n", user.Email, err)
		} else {
			log.Printf("✅ Email jadwal terkirim ke %s\n", user.Email)
		}

		time.Sleep(2 * time.Second)
	}
}
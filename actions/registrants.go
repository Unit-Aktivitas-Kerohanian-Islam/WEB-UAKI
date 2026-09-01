package actions

import (
	"crypto/tls"
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
	q := PaginateFromContext(tx, c)

	// Hanya tampilkan data pendaftar yang sudah submit form pendaftaran (memiliki pilihan divisi)
	baseCondition := "division_1 IS NOT NULL AND TRIM(division_1) != ''"
	q = q.Where(baseCondition)

	// Filter pencarian teks
	search := strings.TrimSpace(c.Param("search"))
	if search == "" {
		search = strings.TrimSpace(c.Param("q"))
	}
	if search != "" {
		searchPattern := "%" + search + "%"
		q = q.Where("(LOWER(COALESCE(name, '')) LIKE LOWER(?) OR LOWER(COALESCE(email, '')) LIKE LOWER(?) OR LOWER(COALESCE(nim, '')) LIKE LOWER(?) OR LOWER(COALESCE(prodi, '')) LIKE LOWER(?) OR LOWER(COALESCE(fakultas, '')) LIKE LOWER(?))", searchPattern, searchPattern, searchPattern, searchPattern, searchPattern)
	}

	// Dukung filter status jika diberikan via query parameter
	status := strings.TrimSpace(c.Param("status"))
	if status != "" && strings.ToUpper(status) != "ALL" {
		if strings.ToUpper(status) == "PENDING" {
			q = q.Where("(status = 'PENDING' OR status IS NULL OR status = '')")
		} else {
			q = q.Where("status = ?", strings.ToUpper(status))
		}
	}

	// Dukung filter divisi jika diberikan via query parameter
	division := strings.TrimSpace(c.Param("division"))
	if division != "" && strings.ToUpper(division) != "ALL" {
		q = q.Where("(division_1 = ? OR division_2 = ?)", division, division)
	}

	q = q.Order("created_at DESC")

	if err := q.All(registrants); err != nil {
		return Response(c, http.StatusInternalServerError, "Failed to retrieve data", err.Error())
	}

	for i := range *registrants {
		(*registrants)[i].Password = nulls.NewString("")
	}

	lolosCount, _ := tx.Where(baseCondition + " AND status = ?", "LOLOS_BERKAS").Count(&models.Registrant{})
	diterimaCount, _ := tx.Where(baseCondition + " AND status = ?", "DITERIMA").Count(&models.Registrant{})
	ditolakCount, _ := tx.Where(baseCondition + " AND status = ?", "DITOLAK").Count(&models.Registrant{})
	pendingCount, _ := tx.Where(baseCondition + " AND (status = ? OR status IS NULL OR status = '')", "PENDING").Count(&models.Registrant{})
	totalSubmitted, _ := tx.Where(baseCondition).Count(&models.Registrant{})
	totalAll := q.Paginator.TotalEntriesSize
	if totalAll == 0 && search == "" && (status == "" || strings.ToUpper(status) == "ALL") && (division == "" || strings.ToUpper(division) == "ALL") {
		totalAll = totalSubmitted
	}

	return Response(c, http.StatusOK, "Success", map[string]interface{}{
		"data":       registrants,
		"pagination": q.Paginator,
		"summary": map[string]interface{}{
			"total":        totalSubmitted,
			"lolos_berkas": lolosCount,
			"pending":      pendingCount,
			"diterima":     diterimaCount,
			"ditolak":      ditolakCount,
		},
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

	if registrant.TwibbonUrl.Valid && registrant.TwibbonUrl.String != "" {
		key := v.storageService.ExtractObjectKey(registrant.TwibbonUrl.String)
		if err := v.storageService.Delete(c.Request().Context(), key); err != nil {
			log.Printf("⚠️ gagal hapus Twibbon lokal: %v", err)
		}
	}

	if registrant.PortofolioUrl.Valid && registrant.PortofolioUrl.String != "" {
		key := v.storageService.ExtractObjectKey(registrant.PortofolioUrl.String)
		if err := v.storageService.Delete(c.Request().Context(), key); err != nil {
			log.Printf("⚠️ gagal hapus Portofolio lokal: %v", err)
		}
	}

	if err := tx.Destroy(registrant); err != nil {
		return Response(c, http.StatusInternalServerError, "Failed to delete", err.Error())
	}

	return Response(c, http.StatusOK, "Deleted successfully", nil)
}

func (v RegistrantsResource) UploadFile(c buffalo.Context) error {
	role, _ := c.Value("role").(string)
	if role != "registrant" {
		return Response(c, http.StatusForbidden, "Hanya Pendaftar yang diizinkan mengunggah berkas", nil)
	}

	c.Request().Body = http.MaxBytesReader(c.Response(), c.Request().Body, 10<<20)

	result := make(map[string]string)

	cvFile, errCV := c.File("cv")
	if errCV == nil {
		defer cvFile.Close()
		ext := strings.ToLower(filepath.Ext(cvFile.Filename))
		if ext != ".pdf" {
			return Response(c, http.StatusBadRequest, "CV harus berupa file PDF (.pdf)", nil)
		}
		safeFilename := strings.ReplaceAll(cvFile.Filename, " ", "_")
		cvURL, err := v.storageService.Upload(c.Request().Context(), "cv/"+uuid.New().String()+"-"+safeFilename, cvFile)
		if err != nil {
			return Response(c, http.StatusInternalServerError, "Gagal menyimpan CV", err.Error())
		}
		result["cv_url"] = cvURL
	}

	twibbonFile, errTwibbon := c.File("twibbon")
	if errTwibbon == nil {
		defer twibbonFile.Close()
		ext := strings.ToLower(filepath.Ext(twibbonFile.Filename))
		if ext != ".jpg" && ext != ".jpeg" && ext != ".png" && ext != ".webp" {
			return Response(c, http.StatusBadRequest, "Twibbon harus berupa file gambar (.jpg, .jpeg, .png, .webp)", nil)
		}
		safeFilename := strings.ReplaceAll(twibbonFile.Filename, " ", "_")
		twibbonURL, err := v.storageService.Upload(c.Request().Context(), "twibbon/"+uuid.New().String()+"-"+safeFilename, twibbonFile)
		if err != nil {
			return Response(c, http.StatusInternalServerError, "Gagal menyimpan Twibbon", err.Error())
		}
		result["twibbon_url"] = twibbonURL
	}

	portofolioFile, errPortofolio := c.File("portofolio")
	if errPortofolio == nil {
		defer portofolioFile.Close()
		ext := strings.ToLower(filepath.Ext(portofolioFile.Filename))
		if ext != ".pdf" {
			return Response(c, http.StatusBadRequest, "Portofolio harus berupa file PDF (.pdf)", nil)
		}
		safeFilename := strings.ReplaceAll(portofolioFile.Filename, " ", "_")
		portofolioURL, err := v.storageService.Upload(c.Request().Context(), "portofolio/"+uuid.New().String()+"-"+safeFilename, portofolioFile)
		if err != nil {
			return Response(c, http.StatusInternalServerError, "Gagal menyimpan Portofolio", err.Error())
		}
		result["portofolio_url"] = portofolioURL
	}

	if len(result) == 0 {
		return Response(c, http.StatusBadRequest, "Setidaknya salah satu file ('cv', 'twibbon', atau 'portofolio') harus disertakan", nil)
	}

	return Response(c, http.StatusOK, "Upload success", result)
}

func (v RegistrantsResource) Login(c buffalo.Context) error {
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

	// PERBAIKAN: Menambahkan kolom Aud untuk menangkap nilai Client ID dari token
	var googleData struct {
		Email string `json:"email"`
		Name  string `json:"name"`
		Aud   string `json:"aud"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&googleData); err != nil {
		return Response(c, http.StatusInternalServerError, "Failed to parse Google response", nil)
	}

	expectedClientID := os.Getenv("GOOGLE_CLIENT_ID")
	if expectedClientID != "" && googleData.Aud != expectedClientID {
		return Response(c, http.StatusUnauthorized, "Token valid, tetapi tidak berasal dari aplikasi UAKI yang sah.", nil)
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

	if input.TwibbonUrl.Valid && input.TwibbonUrl.String != "" && input.TwibbonUrl.String != registrant.TwibbonUrl.String {
		if registrant.TwibbonUrl.Valid && registrant.TwibbonUrl.String != "" {
			key := v.storageService.ExtractObjectKey(registrant.TwibbonUrl.String)
			v.storageService.Delete(c.Request().Context(), key)
		}
		registrant.TwibbonUrl = input.TwibbonUrl
	}

	// Validasi portofolio wajib jika memilih divisi Creative Media (CM)
	isDiv1CM := input.Division1.Valid && strings.ToUpper(strings.TrimSpace(input.Division1.String)) == "CM"
	isDiv2CM := input.Division2.Valid && strings.ToUpper(strings.TrimSpace(input.Division2.String)) == "CM"
	if isDiv1CM || isDiv2CM {
		hasInputPorto := input.PortofolioUrl.Valid && strings.TrimSpace(input.PortofolioUrl.String) != ""
		hasExistingPorto := registrant.PortofolioUrl.Valid && strings.TrimSpace(registrant.PortofolioUrl.String) != ""
		if !hasInputPorto && !hasExistingPorto {
			return Response(c, http.StatusBadRequest, "Portofolio wajib diunggah untuk pendaftar yang memilih Departemen Creative Media (CM)", nil)
		}
	}

	if input.PortofolioUrl.Valid && input.PortofolioUrl.String != "" && input.PortofolioUrl.String != registrant.PortofolioUrl.String {
		if registrant.PortofolioUrl.Valid && registrant.PortofolioUrl.String != "" {
			key := v.storageService.ExtractObjectKey(registrant.PortofolioUrl.String)
			v.storageService.Delete(c.Request().Context(), key)
		}
		registrant.PortofolioUrl = input.PortofolioUrl
	} else if input.PortofolioUrl.Valid && input.PortofolioUrl.String == "" && registrant.PortofolioUrl.Valid && registrant.PortofolioUrl.String != "" {
		key := v.storageService.ExtractObjectKey(registrant.PortofolioUrl.String)
		v.storageService.Delete(c.Request().Context(), key)
		registrant.PortofolioUrl = nulls.NewString("")
	}

	registrant.Name = input.Name
	registrant.Nickname = input.Nickname
	registrant.NIM = input.NIM
	registrant.Angkatan = input.Angkatan
	registrant.Prodi = input.Prodi
	registrant.Fakultas = input.Fakultas
	registrant.Domicile = input.Domicile
	registrant.OriginCity = input.OriginCity
	registrant.SchoolOrigin = input.SchoolOrigin
	registrant.HasRohisExp = input.HasRohisExp
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

func parseScreeningTime(dateStr string) time.Time {
	formats := []string{
		"2006-01-02 15:04:05",
		"2006-01-02 15:04",
		"2006-01-02T15:04:05",
		"2006-01-02T15:04",
		time.RFC3339,
		time.RFC3339Nano,
		"2006-01-02",
	}

	trimmed := strings.TrimSpace(dateStr)
	for _, layout := range formats {
		if t, err := time.ParseInLocation(layout, trimmed, time.Local); err == nil {
			return t
		}
	}
	return time.Now()
}

func formatIndonesianDateTime(t time.Time) string {
	days := []string{"Minggu", "Senin", "Selasa", "Rabu", "Kamis", "Jumat", "Sabtu"}
	months := []string{
		"", "Januari", "Februari", "Maret", "April", "Mei", "Juni",
		"Juli", "Agustus", "September", "Oktober", "November", "Desember",
	}

	dayName := days[int(t.Weekday())]
	monthName := months[int(t.Month())]

	return fmt.Sprintf("%s, %02d %s %d pukul %02d:%02d WIB",
		dayName, t.Day(), monthName, t.Year(), t.Hour(), t.Minute())
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

	parsedDate := parseScreeningTime(input.ScreeningDate)
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

	dateFormatted := formatIndonesianDateTime(parsedDate)
	go sendScheduleEmailsAsync(targets, dateFormatted, input.ScreeningLocation, input.ScreeningLink)

	return Response(c, http.StatusOK, fmt.Sprintf("Proses pengiriman ke %d pendaftar sedang berjalan di latar belakang", len(targets)), nil)
}

func buildScreeningEmailMessage(senderEmail, senderName, toEmail string, user models.Registrant, dateStr, location, link string) []byte {
	subject := "Jadwal Screening Staff Muda UAKI UB 2026"
	cleanLink := strings.TrimSpace(link)
	if cleanLink != "" && !strings.HasPrefix(cleanLink, "http://") && !strings.HasPrefix(cleanLink, "https://") {
		cleanLink = "https://" + cleanLink
	}

	textBody := fmt.Sprintf(`Assalamu'alaikum Warahmatullahi Wabarakatuh,

Halo %s,

Selamat! Berkas pendaftaran kamu untuk Open Recruitment Staff Muda UKM UAKI Universitas Brawijaya 2026 telah lolos seleksi berkas.

Berikut adalah jadwal wawancara / screening kamu:
- Hari / Tanggal: %s
- Lokasi / Ruang : %s
- Tautan Temu   : %s

Catatan Penting:
1. Harap hadir di ruang temu minimal 10 menit sebelum waktu yang ditentukan.
2. Gunakan nama lengkap dan NIM sebagai display name.
3. Berpakaian rapi, sopan, dan menutup aurat.
4. Pastikan koneksi internet, kamera, dan mikrofon berfungsi dengan baik.

Semoga Allah Subhanahu Wa Ta'ala memberikan kemudahan dan kelancaran.

Wassalamu'alaikum Warahmatullahi Wabarakatuh,
Panitia Open Recruitment Staff Muda UAKI UB 2026`, user.Name, dateStr, location, cleanLink)

	htmlBody := fmt.Sprintf(`<!DOCTYPE html>
<html lang="id">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>Jadwal Screening Staff Muda UAKI UB 2026</title>
</head>
<body style="margin:0; padding:0; background-color:#f4f6f5; font-family:'Segoe UI', Tahoma, Geneva, Verdana, sans-serif; color:#2a2a2a;">
  <table role="presentation" width="100%%" cellpadding="0" cellspacing="0" border="0" style="background-color:#f4f6f5; padding:30px 12px;">
    <tr>
      <td align="center">
        <table role="presentation" width="600" cellpadding="0" cellspacing="0" border="0" style="width:100%%; max-width:600px; background-color:#ffffff; border-radius:18px; overflow:hidden; box-shadow:0 8px 24px rgba(0,0,0,0.08); border:1px solid #e2e8e5;">
          <!-- Header Banner -->
          <tr>
            <td style="background:linear-gradient(135deg, #12383b 0%%, #255d61 100%%); padding:36px 30px; text-align:center;">
              <p style="margin:0 0 6px 0; color:#d8e8e6; font-size:12px; font-weight:700; letter-spacing:1.5px; text-transform:uppercase;">
                Unit Aktivitas Kerohanian Islam (UAKI) UB
              </p>
              <h1 style="margin:0; color:#ffffff; font-size:24px; font-weight:800; line-height:1.3;">
                Jadwal Screening Staff Muda 2026
              </h1>
            </td>
          </tr>

          <!-- Body Content -->
          <tr>
            <td style="padding:32px 30px 24px 30px;">
              <p style="margin:0 0 16px 0; font-size:15px; font-weight:600; color:#12383b;">
                Assalamu'alaikum Warahmatullahi Wabarakatuh,
              </p>
              <p style="margin:0 0 20px 0; font-size:14px; line-height:1.6; color:#444444;">
                Halo <strong>%s</strong>, selamat! Berkas pendaftaran kamu telah kami verifikasi dan dinyatakan <span style="color:#059669; font-weight:700;">LOLOS SELEKSI BERKAS</span>.
              </p>

              <!-- Schedule Card -->
              <table role="presentation" width="100%%" cellpadding="0" cellspacing="0" border="0" style="background-color:#f8faf9; border:1px solid #dbe5e1; border-radius:14px; margin-bottom:24px; overflow:hidden;">
                <tr>
                  <td style="padding:16px 20px; border-bottom:1px solid #e8eeeb;">
                    <span style="display:block; font-size:11px; font-weight:700; color:#626f44; text-transform:uppercase; letter-spacing:0.5px;">Waktu & Tanggal</span>
                    <strong style="font-size:15px; color:#12383b;">%s</strong>
                  </td>
                </tr>
                <tr>
                  <td style="padding:16px 20px; border-bottom:1px solid #e8eeeb;">
                    <span style="display:block; font-size:11px; font-weight:700; color:#626f44; text-transform:uppercase; letter-spacing:0.5px;">Lokasi / Ruang</span>
                    <strong style="font-size:14px; color:#2a2a2a;">%s</strong>
                  </td>
                </tr>
                %s
              </table>

              %s

              <!-- Instructions -->
              <div style="background-color:#fefbee; border-left:4px solid #d97706; padding:16px 18px; border-radius:8px; margin-bottom:24px;">
                <p style="margin:0 0 8px 0; font-size:13px; font-weight:700; color:#92400e;">
                  📌 Petunjuk Pelaksanaan Screening:
                </p>
                <ol style="margin:0; padding-left:18px; font-size:12.5px; line-height:1.6; color:#78350f;">
                  <li>Hadir di lokasi / ruang temu minimal <strong>10 menit</strong> sebelum sesi dimulai.</li>
                  <li>Gunakan format nama <code>Nama Lengkap - NIM</code> jika screening dilakukan daring.</li>
                  <li>Berpakaian rapi, sopan, dan menutup aurat.</li>
                  <li>Pastikan koneksi internet, kamera, dan mikrofon berfungsi dengan baik.</li>
                </ol>
              </div>

              <p style="margin:0 0 8px 0; font-size:13px; color:#555555; line-height:1.5;">
                Semoga Allah SWT senantiasa memberikan kelancaran dan kemudahan dalam setiap langkah proses seleksi.
              </p>
              <p style="margin:0; font-size:13px; font-weight:600; color:#12383b;">
                Wassalamu'alaikum Warahmatullahi Wabarakatuh,<br>
                <span style="color:#626f44; font-weight:700;">Panitia OPREC Staff Muda UAKI UB 2026</span>
              </p>
            </td>
          </tr>

          <!-- Footer -->
          <tr>
            <td style="background-color:#edf2f0; padding:18px 30px; text-align:center; border-top:1px solid #e0e7e4;">
              <p style="margin:0; font-size:11px; color:#7a8a87; line-height:1.4;">
                Email ini dikirim secara otomatis oleh Sistem OPREC UKM UAKI Universitas Brawijaya.<br>
                Gedung UKM Lt. 3 Universitas Brawijaya, Malang.
              </p>
            </td>
          </tr>
        </table>
      </td>
    </tr>
  </table>
</body>
</html>`,
		user.Name,
		dateStr,
		location,
		func() string {
			if cleanLink != "" {
				return fmt.Sprintf(`<tr>
                  <td style="padding:16px 20px;">
                    <span style="display:block; font-size:11px; font-weight:700; color:#626f44; text-transform:uppercase; letter-spacing:0.5px;">Tautan Screening</span>
                    <a href="%s" target="_blank" style="color:#2563eb; font-weight:600; font-size:13px; text-decoration:underline; word-break:break-all;">%s</a>
                  </td>
                </tr>`, cleanLink, cleanLink)
			}
			return ""
		}(),
		func() string {
			if cleanLink != "" {
				return fmt.Sprintf(`<div style="text-align:center; margin-bottom:24px;">
                <a href="%s" target="_blank" style="display:inline-block; background-color:#2563eb; color:#ffffff; font-size:14px; font-weight:700; text-decoration:none; padding:12px 28px; border-radius:10px; box-shadow:0 4px 10px rgba(37,99,235,0.25);">
                  Masuk ke Ruang Screening &rarr;
                </a>
              </div>`, cleanLink)
			}
			return ""
		}(),
	)

	boundary := fmt.Sprintf("bnd_%d", time.Now().UnixNano())

	fromHeader := senderEmail
	if senderName != "" {
		fromHeader = fmt.Sprintf("%s <%s>", senderName, senderEmail)
	}

	domain := "gmail.com"
	if parts := strings.Split(senderEmail, "@"); len(parts) == 2 && parts[1] != "" {
		domain = parts[1]
	}
	messageID := fmt.Sprintf("<%d.%s@%s>", time.Now().UnixNano(), uuid.New().String()[:8], domain)
	dateHeader := time.Now().Format(time.RFC1123Z)

	var msg strings.Builder
	msg.WriteString(fmt.Sprintf("From: %s\r\n", fromHeader))
	msg.WriteString(fmt.Sprintf("To: %s\r\n", toEmail))
	msg.WriteString(fmt.Sprintf("Reply-To: %s\r\n", senderEmail))
	msg.WriteString(fmt.Sprintf("Date: %s\r\n", dateHeader))
	msg.WriteString(fmt.Sprintf("Message-ID: %s\r\n", messageID))
	msg.WriteString(fmt.Sprintf("Subject: %s\r\n", subject))
	msg.WriteString("MIME-Version: 1.0\r\n")
	msg.WriteString("X-Mailer: UAKI-Oprec-Mailer/1.0\r\n")
	msg.WriteString("X-Priority: 3 (Normal)\r\n")
	msg.WriteString(fmt.Sprintf("Content-Type: multipart/alternative; boundary=\"%s\"\r\n", boundary))
	msg.WriteString("\r\n")

	// Plain text section
	msg.WriteString(fmt.Sprintf("--%s\r\n", boundary))
	msg.WriteString("Content-Type: text/plain; charset=\"UTF-8\"\r\n")
	msg.WriteString("Content-Transfer-Encoding: 8bit\r\n\r\n")
	msg.WriteString(textBody)
	msg.WriteString("\r\n\r\n")

	// HTML section
	msg.WriteString(fmt.Sprintf("--%s\r\n", boundary))
	msg.WriteString("Content-Type: text/html; charset=\"UTF-8\"\r\n")
	msg.WriteString("Content-Transfer-Encoding: 8bit\r\n\r\n")
	msg.WriteString(htmlBody)
	msg.WriteString("\r\n\r\n")

	msg.WriteString(fmt.Sprintf("--%s--\r\n", boundary))

	return []byte(msg.String())
}

func sendMailRobust(host, port, senderEmail, senderPassword string, toEmail string, msg []byte) error {
	addr := host + ":" + port

	if port == "465" {
		tlsConfig := &tls.Config{
			ServerName: host,
		}
		conn, err := tls.Dial("tcp", addr, tlsConfig)
		if err != nil {
			return fmt.Errorf("gagal koneksi TLS ke %s: %w", addr, err)
		}
		defer conn.Close()

		client, err := smtp.NewClient(conn, host)
		if err != nil {
			return fmt.Errorf("gagal inisialisasi SMTP client: %w", err)
		}
		defer client.Quit()

		if senderEmail != "" && senderPassword != "" {
			auth := smtp.PlainAuth("", senderEmail, senderPassword, host)
			if err := client.Auth(auth); err != nil {
				return fmt.Errorf("gagal autentikasi SMTP: %w", err)
			}
		}

		if err := client.Mail(senderEmail); err != nil {
			return fmt.Errorf("gagal set MAIL FROM: %w", err)
		}
		if err := client.Rcpt(toEmail); err != nil {
			return fmt.Errorf("gagal set RCPT TO: %w", err)
		}

		w, err := client.Data()
		if err != nil {
			return fmt.Errorf("gagal inisialisasi data writer: %w", err)
		}
		if _, err := w.Write(msg); err != nil {
			return fmt.Errorf("gagal menulis isi email: %w", err)
		}
		return w.Close()
	}

	auth := smtp.PlainAuth("", senderEmail, senderPassword, host)
	return smtp.SendMail(addr, auth, senderEmail, []string{toEmail}, msg)
}

func sendScheduleEmailsAsync(targets []models.Registrant, dateStr, location, link string) {
	senderEmail := strings.TrimSpace(os.Getenv("SMTP_EMAIL"))
	senderPassword := strings.TrimSpace(os.Getenv("SMTP_PASSWORD"))
	smtpHost := strings.TrimSpace(os.Getenv("SMTP_HOST"))
	smtpPort := strings.TrimSpace(os.Getenv("SMTP_PORT"))
	senderName := strings.TrimSpace(os.Getenv("SMTP_SENDER_NAME"))

	// Sanitize Google App Password if it contains spaces (e.g. "upmx fipw bibs wzki" -> "upmxfipwbibswzki")
	if smtpHost == "smtp.gmail.com" || strings.HasSuffix(strings.ToLower(senderEmail), "@gmail.com") {
		senderPassword = strings.ReplaceAll(senderPassword, " ", "")
	}

	if smtpHost == "" {
		smtpHost = "smtp.gmail.com"
	}
	if smtpPort == "" {
		smtpPort = "587"
	}
	if senderName == "" {
		senderName = "Panitia OPREC Staff Muda UAKI UB"
	}

	if senderEmail == "" || senderPassword == "" {
		log.Println("⚠️ [SMTP] SMTP_EMAIL atau SMTP_PASSWORD kosong. Pengiriman email screening dibatalkan.")
		return
	}

	log.Printf("📧 [SMTP] Memulai pengiriman jadwal ke %d pendaftar via %s:%s (Pengirim: %s)...\n", len(targets), smtpHost, smtpPort, senderEmail)

	for _, user := range targets {
		if strings.TrimSpace(user.Email) == "" {
			log.Printf("⚠️ [SMTP] Melewati pendaftar %s (ID: %s) karena email kosong.\n", user.Name, user.ID)
			continue
		}

		msg := buildScreeningEmailMessage(senderEmail, senderName, user.Email, user, dateStr, location, link)

		err := sendMailRobust(smtpHost, smtpPort, senderEmail, senderPassword, user.Email, msg)
		if err != nil {
			log.Printf("❌ [SMTP] Gagal kirim email ke %s (%s): %v\n", user.Name, user.Email, err)
			if strings.Contains(err.Error(), "535") || strings.Contains(err.Error(), "BadCredentials") {
				log.Println("💡 [SMTP Hint] Untuk akun Gmail/Google Workspace, gunakan 'App Password' (16 karakter) dari Akun Google -> Keamanan -> Sandi Aplikasi, bukan password login biasa.")
			}
		} else {
			log.Printf("✅ [SMTP] Email jadwal screening berhasil dikirim ke %s (%s)\n", user.Name, user.Email)
		}

		time.Sleep(1 * time.Second)
	}
}
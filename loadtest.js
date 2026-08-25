import http from 'k6/http';
import { check, sleep, group } from 'k6';

// Konfigurasi Beban Ekstrem: Simulasi "Deadline Penutupan OPREC UAKI"
export const options = {
  stages: [
    { duration: '30s', target: 50 },  // Pemanasan: 50 user masuk web
    { duration: '1m', target: 200 },  // PUNCAK: Tiba-tiba 200 user submit barengan (Rush Hour)
    { duration: '30s', target: 200 }, // Tahan badai 200 user selama 30 detik
    { duration: '30s', target: 0 },   // Pendaftaran ditutup, user turun ke 0
  ],
  thresholds: {
    // Karena ini badai ekstrem, kita toleransi error maksimal 5%
    http_req_failed: ['rate<0.05'], 
    // Saat badai, server pasti melambat. 95% request boleh selesai di bawah 2 detik (2000ms)
    http_req_duration: ['p(95)<2000'], 
  },
};

const BASE_URL = 'http://127.0.0.1:3000';
const REGISTRANT_TOKEN = __ENV.REGISTRANT_TOKEN;
const ADMIN_TOKEN = __ENV.ADMIN_TOKEN;

// Dummy file PDF di memori untuk test endpoint Upload CV
const dummyPDF = http.file('%PDF-1.4 \n Dummy content for load testing', 'cv_dummy.pdf', 'application/pdf');

export default function () {
  
  // 1. SKENARIO PUBLIK (Pengunjung web melihat-lihat info/artikel)
  group('Endpoint Publik', () => {
    let resArticles = http.get(`${BASE_URL}/articles`);
    check(resArticles, { 'GET /articles is 200': (r) => r.status === 200 });

    let resMedia = http.get(`${BASE_URL}/media`);
    check(resMedia, { 'GET /media is 200': (r) => r.status === 200 });

    let resCategory = http.get(`${BASE_URL}/media-categories`);
    check(resCategory, { 'GET /media-categories is 200': (r) => r.status === 200 });
  });

  // 2. SKENARIO PENDAFTAR (Harus sedia REGISTRANT_TOKEN)
  if (REGISTRANT_TOKEN) {
    group('Endpoint Pendaftar', () => {
      let headers = { 'Authorization': `Bearer ${REGISTRANT_TOKEN}` };

      // A. Load Data Profil
      let resMe = http.get(`${BASE_URL}/registrants/me`, { headers: headers });
      check(resMe, { 'GET /me is 200': (r) => r.status === 200 });

      // B. Upload File CV
      let resUpload = http.post(`${BASE_URL}/registrants/cv`, { cv: dummyPDF }, { headers: headers });
      check(resUpload, { 'POST /cv is 200': (r) => r.status === 200 });

      let cvUrl = "";
      if (resUpload.status === 200) {
         cvUrl = resUpload.json('url') || "";
      }

      // C. Submit Form Profil Lengkap
      let payload = JSON.stringify({
        name: "M. Ilham Akbar Priatama",
        nim: "235150401111059",
        angkatan: "2023",
        prodi: "Sistem Informasi",
        fakultas: "Fakultas Ilmu Komputer",
        domicile: "Malang",
        phone: "081234567890",
        division_1: "KP",
        division_2: "HUMAS",
        swot_s: "Pemahaman Backend & QA",
        swot_w: "Manajemen Waktu",
        swot_o: "Banyak project implementasi",
        swot_t: "Tugas kuliah menumpuk",
        organization_exp: "Kopegtel Malang, Diskominfo",
        commitment: "Berkomitmen penuh berkontribusi",
        cv_url: cvUrl
      });

      headers['Content-Type'] = 'application/json';
      let resUpdate = http.put(`${BASE_URL}/registrants/me`, payload, { headers: headers });
      check(resUpdate, { 'PUT /me is 200': (r) => r.status === 200 });
    });
  }

  // 3. SKENARIO ADMIN (Harus sedia ADMIN_TOKEN)
  if (ADMIN_TOKEN) {
    group('Endpoint Admin', () => {
      let headers = {
        'Authorization': `Bearer ${ADMIN_TOKEN}`,
        'Content-Type': 'application/json',
      };

      // A. Cek Daftar Peserta OPREC
      let resRegList = http.get(`${BASE_URL}/registrants?page=1&per_page=10`, { headers: headers });
      check(resRegList, { 'GET /registrants list is 200': (r) => r.status === 200 });

      // B. Cek Daftar Admin Server
      let resAdminList = http.get(`${BASE_URL}/admins`, { headers: headers });
      check(resAdminList, { 'GET /admins list is 200': (r) => r.status === 200 });
    });
  }

  // Jeda 1 detik antar aksi agar tidak dianggap serangan DDoS/Spam agresif
  sleep(1);
}
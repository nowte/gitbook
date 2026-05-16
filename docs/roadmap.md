# gitBook GO — Roadmap

**Son Güncelleme:** 2026-05-13
**Mevcut Sürüm:** v1.02.01
**Hedef:** Production-ready, küçük-orta ekipler için güvenli ve kullanışlı Git arayüzü

> Bu yol haritası system-audit-report.md bulgularına ve öncelik sıralamasına göre hazırlanmıştır.
> Her sürüm önceki sürüm üzerine inşa edilir; kritik güvenlik düzeltmeleri her zaman önce gelir.

---

## Durum Simgeleri

```
[DONE]     Tamamlandi
[WIP]      Gelistirme sureci
[PLANNED]  Planlandı, henüz baslanmadi
[IDEA]     Gelecekte degerlendirilecek
```

---

## v1.03.00 — Stabilite & Guvenlik Temeli
**Hedef Tarih:** 2026-07 (tahmini 6-8 hafta)
**Tema:** Audit raporundaki KRITIK ve YUKSEK öncelikli sorunlari kapatmak

### [DONE] Concurrent Access Koruması
- `sync.Mutex` ile git write operasyonlarini serialize etme
- `index.lock` hataları için otomatik retry mekanizması
- Config dosyasi yazmada atomic rename
- Birden fazla gitBook instance'ı ayni repoda calıştıgında uyari (advisory PID lock)

### [DONE] Rollback / Snapshot Sistemi
- Her destructive islem öncesi otomatik snapshot alma
  (`reset --hard`, `rebase`, `merge`)
- `.gitbook/snapshots/` altında hafif JSON snapshot kaydi
  (commit hash + branch state + staged files listesi)
- `/undo` komutu: son snapshot'a tek adımda dönüş
- `/snapshots` komutu: tüm snapshot'ları listele
- Maksimum 10 snapshot tutulur, eskiler otomatik temizlenir

### [DONE] Geliştirilmiş Error Handling
- Panic recovery middleware (`recover()` ile graceful degradation)
- Git timeout aşımında açıklayıcı mesaj + `/status` önerisi
- Boş repo, yok edilmis HEAD, detached HEAD için özel mesajlar
- Multi-instance çakışma uyarısı

### [PLANNED] Temel Test Altyapısı
- `internal/git` paketi için unit testler (mock git runner)
- `internal/config` paketi için unit testler
- CI workflow tanımı (GitHub Actions)
- Hedef: %60 code coverage kritik paketlerde

### Degisecek Dosyalar
```
internal/git/adapter.go       -- mutex + retry + timeout fix  [DONE]
internal/git/snapshot.go      -- YENI: snapshot yöneticisi    [DONE]
internal/config/config.go     -- mutex + atomic write + instance lock [DONE]
internal/ui/handlers.go       -- snapshot hooks               [DONE]
internal/ui/handlers_undo.go  -- YENI: /undo + /snapshots     [DONE]
internal/ui/types.go          -- panic recovery middleware     [DONE]
internal/ui/chat.go           -- genişletilmiş hata hint'leri [DONE]
internal/lang/lang_en.go      -- yeni dil anahtarları         [DONE]
internal/lang/lang_tr.go      -- yeni dil anahtarları         [DONE]
cmd/gitbook/main.go           -- instance lock entegrasyonu   [DONE]
*_test.go                     -- PLANNED: test dosyaları
```

---

## v1.04.00 — Kullanici Deneyimi & Canli Önizleme
**Hedef Tarih:** 2026-09 (tahmini 4-5 hafta)
**Tema:** UX sorunlarını çözmek, görsel geri bildirimi güçlendirmek

### [PLANNED] Live Preview Sistemi
- `/review` ekranında staged/unstaged diff'i renkli göster
  (eklenen satirlar yesil `+`, silinen satirlar kırmızı `-`)
- Commit öncesi otomatik diff özeti (satır/dosya sayısı)
- `/stage` sonrasinda anlık staged dosya listesi

### [PLANNED] Interaktif Staging
- Dosya listesi üzerinden seçimli staging (checkbox benzeri TUI)
- `/stage` ile dosya tarayıcısı entegrasyonu
- `git add -p` benzeri hunk-bazlı seçim (basit versiyon)

### [PLANNED] Gelişmiş `/log` Ekranı
- Commit grafiği (ASCII art, branch splits görünür)
- Commit detayına tıklama: diff, author, date
- Sayfalama (q ile çıkış, j/k ile gezinme)

### [PLANNED] Komut Geçmişi
- Ok tuşları ile önceki komutlara dönme (shell history benzeri)
- `.gitbook/history` dosyasına yazma (max 200 satır)
- `/history` komutu ile listeleme

### Degisecek Dosyalar
```
internal/ui/handlers.go       -- review + log iyilestirmeleri
internal/ui/chat.go           -- history buffer
internal/ui/home.go           -- staging TUI
internal/git/adapter.go       -- diff formatting helpers
```

---

## v1.05.00 — Conflict Resolution & Multi-user Temeli
**Hedef Tarih:** 2026-11 (tahmini 6-8 hafta)
**Tema:** Takim calısmasını kolaylastırmak, conflict sorunlarını çözmek

### [PLANNED] Gelişmiş Conflict Resolver
- Merge/rebase conflict'lerini tespit edip açıklayıcı mesaj göster
- Her conflict dosyasını listele, konumunu belirt (satır no)
- `<<<<<<< HEAD ... ======= ... >>>>>>>` bloklarını renklendir
- Çözüm sonrası otomatik `/stage` + commit öneri akışı

### [PLANNED] Branch Karşılaştırma
- `/diff <branch>` çıktısını dosya listesi + değişiklik özeti olarak göster
- İki branch arasındaki commit farkını listeleme (`/log main..feature/X`)
- Merge öncesi "bu branch'te ne var?" özet ekranı

### [PLANNED] Temel Multi-user Farkındalık
- Aynı branch'te başka biri çalışıyorsa uyarı (upstream kontrolü)
- Push öncesi "birisi daha commit etti" bildirimi
- Remote durum cache'i (5 dakikada bir otomatik fetch)

### [PLANNED] Conflict Önleme Tavsiyeleri
- Uzun süredir güncellenmemiş feature branch'lere uyarı
- "Bu branch 7 gündür main'den uzaklaştı, rebase önerilir" bildirimi

### Degisecek Dosyalar
```
internal/git/adapter.go         -- conflict detection helpers
internal/ui/handlers.go         -- conflict flow
internal/ui/smart_handlers.go   -- branch comparison
internal/smart/conflict.go      -- YENI: conflict analiz motoru
```

---

## v1.06.00 — Performans & Offline Destek
**Hedef Tarih:** 2027-01 (tahmini 4-5 hafta)
**Tema:** Büyük repolar + offline senaryolarda güvenilirlik

### [PLANNED] Offline Mode
- Network yoksa read-only mod (status, log, diff çalısır)
- Offline'da yapılan staging/commit'leri kuyruğa al
- Bağlantı geri gelince otomatik sync teklifi

### [PLANNED] Bellek ve Performans Optimizasyonu
- Büyük diff çıktılarını sayfalı göster (scroll)
- Git log'u lazy-load (ilk 20 commit, kaydırunca +20 daha)
- Status cache: 2 saniye içinde ikinci `/status` çağrısı cache'den döner

### [PLANNED] Büyük Repo Desteği
- 10.000+ dosyalı repolar için `/status` hız iyilestirmesi
- `.gitbook/cache/` altında branch ve remote bilgisi önbellekleme
- Monorepo için alt-dizin bazlı çalışma modü (`/cd` ile)

---

## v1.07.00 — Güvenlik Sertleştirmesi
**Hedef Tarih:** 2027-02 (tahmini 3-4 hafta)
**Tema:** Audit raporundaki güvenlik önerilerini eksiksiz kapatmak

### [PLANNED] Hassas Veri Şifreleme
- `.gitbook/profiles.json` içindeki token/key alanlarını at-rest şifrele
- Bellek içi credential temizleme (sıfırlama sonrası sıfır-out)

### [PLANNED] Gelişmiş Audit Log
- Log dosyalarını JSON Lines formatına taşı (mevcut plain text yerine)
- Log rotation: günlük dosya, maksimum 30 gün saklama
- `/audit` komutu: son N log satırını terminalde göster
- Kritik olaylar için ayrı `security.log` dosyası

### [PLANNED] Rate Limiting & Abuse Koruması
- Saniyede 10'dan fazla komut girilirse yavaşlatma
- Tekrarlayan başarısız push denemelerinde otomatik bekleme

---

## v1.08.00 — Plugin Mimarisi (Ekosistem)
**Hedef Tarih:** 2027-04 (tahmini 8-10 hafta)
**Tema:** Topluluğun katkı yapabilmesi, özelleştirilebilirlik

### [IDEA] Plugin Sistemi
- `.gitbook/plugins/` dizininden harici komut yükleme
- Plugin API: komut kaydı, dil anahtarı ekleme, hook noktaları
- Resmi plugin örnekleri: `gitflow`, `conventional-commits-strict`, `jira-linker`

### [IDEA] Hook Sistemi
- Pre-commit, post-commit, pre-push hook'ları
- Shell script veya Go plugin olarak tanımlanabilir
- `/hook list` ve `/hook add <event> <script>` komutları

### [IDEA] Theme Sistemi
- `~/.gitbook/theme.json` ile renk özelleştirme
- Hazır temalar: `default`, `minimal`, `high-contrast`, `solarized`

---

## v2.00.00 — Büyük Ekip & Enterprise Temeli
**Hedef Tarih:** 2027-06+ (uzun vadeli)
**Tema:** Audit raporundaki enterprise ve büyük ekip eksikliklerini kapatmak

### [IDEA] Role-Based Access Control
- `.gitbook/team.json` ile kullanıcı rolleri (owner, dev, readonly)
- Rol bazlı komut kısıtlamaları (readonly kullanici push yapamaz)

### [IDEA] GUI Alternatifi
- Electron veya Tauri tabanlı masaüstü uygulama (terminal gerektirmeyen)
- Aynı backend, farklı UI katmanı

### [IDEA] CI/CD Entegrasyonu
- GitHub Actions için `gitbook-action` resmi action'ı
- GitLab CI için `.gitlab-ci.yml` şablonu
- Commit kuralları ihlalini CI'da hata olarak raporlama

### [IDEA] Cloud Sync
- `.gitbook/` konfigürasyonunu bulut üzerinde takım içinde paylaşma
- Profil ve kural senkronizasyonu

---

## Versiyon Bagimliliklari

```
v1.02.01 (mevcut)
    |
    v
v1.03.00  [Stabilite & Guvenlik]  <-- KRITIK, atlanamazaz
    |
    v
v1.04.00  [UX & Önizleme]
    |
    v
v1.05.00  [Conflict & Multi-user]
    |
    v
v1.06.00  [Performans & Offline]
    |
    v
v1.07.00  [Güvenlik Sertlestirmesi]
    |
    v
v1.08.00  [Plugin Mimarisi]
    |
    v
v2.00.00  [Enterprise]
```

---

## Hedef Kitle Evrimi

| Surüm | Hedef Kitle |
|-------|-------------|
| v1.02.01 (simdi) | Tek gelistiriciler, Git ögrenenler |
| v1.03.00 — v1.04.00 | Tek gelistiriciler + 2-3 kisilik kücük ekipler |
| v1.05.00 — v1.06.00 | 5-10 kisilik orta ekipler |
| v1.07.00 — v1.08.00 | Kücük-orta ekipler, plugin geliscitrenleri |
| v2.00.00+ | Büyük ekipler, enterprise |

---

## Katkida Bulunmak

Yol haritasindaki bir özellik üzerinde çalışmak istiyorsanız:
1. Önce ilgili versiyonun issue'sunu açın
2. Tasarım kararlarını tartısın (breaking change içeriyorsa)
3. PR açmadan önce `_test.go` dosyaları ekleyin
4. Dil anahtarlarını hem `lang_tr.go` hem `lang_en.go`'ya ekleyin

GitHub: https://github.com/nowte/gitbook

---

*Rapor kaynağı: docs/system-audit-report.md (Audit Tarihi: 2026-05-12)*
*Sonraki roadmap güncellemesi: v1.03.00 release sonrası*

---

## v1.03.01 — Intent Language & UX Overhaul
**Tarih:** 2026-05-16
**Tema:** Tüm kullanıcı arayüz dilini git jargonundan niyet diline taşıma

### [DONE] lang_en.go — Tam Niyet Dili Yeniden Yazımı
- Tüm komut açıklamaları "commit/stage/branch/upstream" yerine niyet tabanlı dile çevrildi
- Tüm hata mesajları kullanıcı dostu hale getirildi
- Auto-pipeline adım metinleri teknik değil insan dilinde
- Wizard prompt'ları sadeleştirildi

### [DONE] lang_tr.go — Türkçe Tam Uyum
- lang_en.go ile tam eşdeğer niyet dili uygulaması
- Türkçe doğal dil akışı korundu

### [DONE] types.go — Komut Listesi Açıklamaları
- Komut listesindeki tüm `desc:` alanları niyet diline çevrildi
- Wizard field label'ları git jargonundan arındırıldı ("Commit message" → "Save description")
- Wizard placeholder'ları somut örneklerle güncellendi

### [DONE] home.go — Başlangıç Ekranı
- Tip satırı: `/init` yerine `/tutorial` önerildi (yeni kullanıcıya daha uygun)
- Placeholder intent-based dile çevrildi

### [DONE] chat.go — Yardım Metni
- Help komut argüman etiketleri git jargonundan arındırıldı
  (`/diff [branch]` → `/diff [workspace]`)

### [DONE] docs/ux-design-principles.md — YENİ BELGE
- Tüm niyet-komut eşleştirme tablosu
- Yasaklı kelimeler listesi ve alternatifleri
- Hata mesajı yazım kuralları
- UX akış kuralları (5 kural)
- Çeviri sistemi kılavuzu
- Görsel tasarım token'ları


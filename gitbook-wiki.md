# gitBook — Kullanım Kılavuzu

> Git bilmene gerek yok. Bu kılavuz gitBook'un tüm özelliklerini sıfırdan anlatır.

---

## İçindekiler

1. [Kurulum ve İlk Başlangıç](#1-kurulum-ve-i̇lk-başlangıç)
2. [Sıfırdan GitHub'a Proje Yüklemek](#2-sıfırdan-githuba-proje-yüklemek)
3. [Var Olan Projeye Güncelleme Atmak](#3-var-olan-projeye-güncelleme-atmak)
4. [GitHub'dan Proje İndirmek](#4-githubdan-proje-i̇ndirmek)
5. [Çalışma Alanları (Branch)](#5-çalışma-alanları-branch)
6. [Geçmişe Dönmek ve Geri Almak](#6-geçmişe-dönmek-ve-geri-almak)
7. [Güncelleme Çakışmaları](#7-güncelleme-çakışmaları)
8. [Bitmemiş Değişiklikleri Bir Kenara Bırakmak](#8-bitmemiş-değişiklikleri-bir-kenara-bırakmak)
9. [Versiyon İşaretlemek (Tag)](#9-versiyon-i̇şaretlemek-tag)
10. [Akıllı Özellikler](#10-akıllı-özellikler)
11. [Tüm Komutlar — Hızlı Referans](#11-tüm-komutlar--hızlı-referans)
12. [Hata Mesajları Rehberi](#12-hata-mesajları-rehberi)

---

## 1. Kurulum ve İlk Başlangıç

### Kimliğini tanıt

gitBook'u ilk açtığında ya da yeni bir bilgisayarda kullanmaya başladığında kimliğini bir kez girmen gerekir. Bu bilgiler kayıtlarında görünür.

```
/setup
```

Sırası ile şunları sorar:

| Adım | Ne yazacaksın | Örnek |
|------|---------------|-------|
| Adın ve soyadın | Gerçek adın | `Ahmet Yılmaz` |
| E-posta adresin | GitHub'da kullandığın e-posta | `ahmet@ornek.com` |

> Yanlış girdiysen istediğin zaman `/setup` yazarak tekrar değiştirebilirsin.

### Doğru klasörde olduğunu kontrol et

```
/pwd
```

Hangi klasörde olduğunu gösterir. Projen nerede duruyorsa oraya geçmen gerekir.

```
/cd /Users/ahmet/Projeler/benim-projem
```

> İstersen `/path` yazarak görsel klasör seçici açabilirsin.

---

## 2. Sıfırdan GitHub'a Proje Yüklemek

Bu en çok ihtiyaç duyulan senaryodur. Bilgisayarında bir klasör var, GitHub'a taşımak istiyorsun.

### Adım adım:

**1. Proje klasörüne git**

```
/cd /Users/ahmet/Projeler/benim-projem
```

**2. Versiyon takibini aç**

```
/init
```

Bu klasör artık takip altına girdi. Henüz GitHub'da hiçbir şey yok.

**3. Kimliğini tanıt (ilk kez kullanıyorsan)**

```
/setup
```

**4. Dosyaları kaydedilmeye hazırla**

```
/stage
```

Tüm dosyaları işaretler. Sadece belirli bir dosyayı işaretlemek için:

```
/stage src/main.go
```

**5. İlk kaydı yap**

```
/commit İlk kayıt
```

Ne yaptığını birkaç kelimeyle açıkla. Bu açıklama geçmişinde görünür.

**6. GitHub'da yeni bir repo oluştur**

GitHub.com'a git → "New repository" → Boş repo oluştur (README ekleme) → URL'yi kopyala.

Örnek URL: `https://github.com/ahmet/benim-projem.git`

**7. Projeyi GitHub'a bağla**

```
/github https://github.com/ahmet/benim-projem.git
```

**8. GitHub'a gönder**

```
/push
```

Hepsi bu kadar. Projen artık GitHub'da.

---

## 3. Var Olan Projeye Güncelleme Atmak

Projen zaten GitHub'a bağlı, değişiklik yaptın ve güncel sürümü göndermek istiyorsun.

### Her gün yapacağın akış:

**1. Değişiklikleri kontrol et**

```
/status
```

Hangi dosyaların değiştiğini, neyin kaydedilmeye hazır olduğunu gösterir.

**2. Değişiklikleri gözden geçir (isteğe bağlı)**

```
/review
```

Tam olarak ne değişti, satır satır görmek için:

```
/diff
```

**3. Dosyaları işaretle**

```
/stage
```

Tüm değişiklikler için. Sadece bir klasörü işaretlemek için:

```
/stage src/
```

**4. Kaydet**

```
/commit Giriş sayfası tasarımı güncellendi
```

Açıklama ne kadar net olursa geçmişte o kadar kolay bulursun.

**5. GitHub'a gönder**

```
/push
```

### Önce GitHub'dan güncelleme indir

Başka biri de aynı projede çalışıyorsa ya da farklı bir bilgisayardan değişiklik yaptıysan, göndermeden önce indirmen gerekir:

```
/pull
```

Sonra normal akışa devam et: `/stage` → `/commit` → `/push`

---

## 4. GitHub'dan Proje İndirmek

Başkasına ait ya da başka bilgisayardan oluşturduğun bir projeyi indirmek için:

```
/clone https://github.com/ahmet/benim-projem.git
```

Farklı bir klasör adıyla indirmek için:

```
/clone https://github.com/ahmet/benim-projem.git yeni-klasor-adi
```

Proje indirilir ve otomatik olarak o klasöre girilir.

---

## 5. Çalışma Alanları (Branch)

Çalışma alanları, ana projeyi bozmadan yeni bir özellik denemek için kullanılır. Bitince ana projeyle birleştirilir.

### Mevcut çalışma alanlarını listele

```
/branch
```

### Yeni çalışma alanı aç

```
/start odeme-sayfasi
```

`odeme-sayfasi` adında yeni bir çalışma alanı açıldı. Artık burada yaptığın değişiklikler ana projeyi etkilemez.

### Çalışmayı bitir ve birleştir

```
/finish
```

Çalışma alanındaki değişiklikler ana projeyle birleştirilir.

### Tamamlanan çalışma alanını temizle

```
/cleanup odeme-sayfasi
```

### Başka bir çalışma alanına geç

```
/branch main
```

> `main` yerine geçmek istediğin alanın adını yaz.

---

## 6. Geçmişe Dönmek ve Geri Almak

### Kayıt geçmişini gör

```
/log
```

Son 10 kaydı gösterir. Daha fazlası için:

```
/log 25
```

### Son kaydı geri al (değişiklikleri koru)

```
/reset 1
```

Son 1 kaydı geri alır ama dosyalardaki değişiklikler durmaya devam eder. `1` yerine daha büyük bir sayı yazabilirsin.

### Son kaydı geri al (değişiklikleri sil) — DİKKATLİ

```
/reset-hard 1
```

Bu işlem geri alınamaz. Onay ister: `confirm` yazman gerekir.

> gitBook yıkıcı işlemlerden önce otomatik kontrol noktası kaydeder. Bir şeyler ters giderse `/undo` ile kurtarabilirsin.

### Geçmişteki belirli bir kaydı iptal et

```
/log
```

ile kayıt ID'sini bul (örn. `a3f8c12`), sonra:

```
/revert a3f8c12
```

Bu kayıt güvenle geri alınır, geri kalanlar korunur.

### Son kontrol noktasına dön

```
/undo
```

---

## 7. Güncelleme Çakışmaları

Sen ve başkası aynı satırı değiştirdiyseniz gitBook hangisini seçeceğini bilemez, senden karar vermeni ister.

Çakışma mesajı görürsen:

1. Belirtilen dosyayı metin editöründe aç
2. `<<<<<<<`, `=======`, `>>>>>>>` işaretli bölümleri bul
3. Hangisini tutmak istiyorsan onu bırak, diğerini sil (işaretleri de sil)
4. Kaydet

Sonra gitBook'a dön:

```
/stage
/commit Çakışma giderildi
```

---

## 8. Bitmemiş Değişiklikleri Bir Kenara Bırakmak

Bir şey üzerinde çalışıyorsun ama henüz kaydetmeden başka bir şeye bakman gerekiyor.

```
/stash
```

İsteğe bağlı not ekleyebilirsin:

```
/stash giriş formu yarım kaldı
```

Kenarda bekleyenleri gör:

```
/stash-list
```

Geri getir:

```
/stash-pop
```

---

## 9. Versiyon İşaretlemek (Tag)

Projenin önemli bir aşamasını işaretlemek için kullanılır. Örn. `v1.0.0`.

### Versiyon işaretle

```
/tag v1.0.0
```

### Mevcut işaretleri listele

```
/tag
```

### İşaretleri GitHub'a gönder

```
/tag-push
```

> Normal `/push` versiyon işaretlerini göndermez. Bunun için ayrıca `/tag-push` gerekir.

---

## 10. Akıllı Özellikler

### Değişiklik analizi

Ne değiştirdiğini, kaç dosyayı etkilediğini, hangi türde değişiklikler olduğunu görmek için:

```
/analyze
```

### Kayıt açıklaması öner

Ne yazdığını AI analiz eder ve uygun bir açıklama önerir:

```
/suggest
```

Beğenirsen önerilen açıklamayı `/commit` ile kullanabilirsin.

### Yoksayma listesi oluştur

Geçici dosyaları, derlenmiş çıktıları, API anahtarlarını vs. otomatik olarak takip dışında bırakır:

```
/gitignore
```

Önce önizlemek için:

```
/gitignore preview
```

### Profil yönetimi

İş ve kişisel projeler için farklı kimlik bilgileri kullanmak isteyenler için:

```
/profile set work
/profile set personal
/profile show
```

---

## 11. Tüm Komutlar — Hızlı Referans

### Kurulum

| Komut | Ne yapar |
|-------|----------|
| `/setup` | Adını ve e-postanı gir |
| `/config` | Mevcut kimlik ve proje bilgilerini göster |
| `/language tr` | Dili Türkçe yap |
| `/language en` | Dili İngilizce yap |
| `/info` | Uygulama ve proje bilgileri |

### Klasör

| Komut | Ne yapar |
|-------|----------|
| `/pwd` | Hangi klasörde olduğunu göster |
| `/cd /yol/klasor` | Klasör değiştir |
| `/path` | Görsel klasör seçici |

### Proje

| Komut | Ne yapar |
|-------|----------|
| `/init` | Bu klasörde versiyon takibini başlat |
| `/status` | Değişikliklere genel bakış |
| `/log` | Kayıt geçmişi |
| `/log 25` | Son 25 kaydı göster |
| `/review` | Kaydedilmeye hazır değişiklikleri önizle |
| `/diff` | Değişiklikleri satır satır gör |
| `/diff main` | Başka bir çalışma alanıyla karşılaştır |
| `/blame dosya.txt` | Her satırı en son kim değiştirdi |

### Kaydetmek

| Komut | Ne yapar |
|-------|----------|
| `/stage` | Tüm değişiklikleri işaretle |
| `/stage src/main.go` | Belirli dosyayı işaretle |
| `/unstage src/main.go` | Dosyayı işaret listesinden çıkar |
| `/commit Açıklama` | İşaretlenen değişiklikleri kaydet |
| `/amend Yeni açıklama` | Son kaydın açıklamasını düzelt |

### GitHub

| Komut | Ne yapar |
|-------|----------|
| `/github https://...` | Projeyi GitHub'a bağla |
| `/remote` | Bağlı olduğun adresi göster |
| `/push` | Kayıtları GitHub'a gönder |
| `/pull` | GitHub'dan değişiklikleri indir |
| `/fetch` | GitHub'da ne var kontrol et (indirmez) |
| `/sync` | Kaç kayıt ileride/geride olduğunu gör |
| `/clone https://...` | GitHub'dan proje indir |

### Çalışma Alanları

| Komut | Ne yapar |
|-------|----------|
| `/branch` | Tüm çalışma alanlarını listele |
| `/branch main` | `main` alanına geç |
| `/start özellik-adı` | Yeni çalışma alanı aç |
| `/finish` | Çalışma alanını ana projeyle birleştir |
| `/cleanup alan-adı` | Tamamlanan alanı sil |

### Geri Almak

| Komut | Ne yapar |
|-------|----------|
| `/reset 1` | Son 1 kaydı geri al (dosyalar korunur) |
| `/reset-hard 1` | Son 1 kaydı sil (değişiklikler de gider) |
| `/revert a3f8c12` | Geçmişteki kaydı güvenle iptal et |
| `/cherry-pick a3f8c12` | Geçmişteki bir kaydı buraya uygula |
| `/undo` | Son kontrol noktasına dön |
| `/rebase main` | Çalışma alanını `main` üzerine taşı |

### Bir Kenara Bırakmak

| Komut | Ne yapar |
|-------|----------|
| `/stash` | Bitmemiş değişiklikleri geçici sakla |
| `/stash not` | Notla birlikte sakla |
| `/stash-list` | Saklananları listele |
| `/stash-pop` | Son saklananı geri getir |

### Versiyonlamak

| Komut | Ne yapar |
|-------|----------|
| `/tag` | Mevcut işaretleri listele |
| `/tag v1.0.0` | Versiyon işareti koy |
| `/tag-push` | İşaretleri GitHub'a gönder |

### Akıllı Araçlar

| Komut | Ne yapar |
|-------|----------|
| `/analyze` | Değişiklikleri analiz et |
| `/suggest` | Kayıt açıklaması öner |
| `/gitignore` | Yoksayma listesi oluştur |
| `/gitignore preview` | Önce önizle |
| `/profile show` | Aktif profili göster |
| `/profile set work` | İş profiline geç |

### Diğer

| Komut | Ne yapar |
|-------|----------|
| `/help` | Tüm komutları listele |
| `/tutorial` | Başlangıç rehberi |
| `/new` | Ekranı temizle |
| `/exit` | Uygulamadan çık |

---

## 12. Hata Mesajları Rehberi

| Mesaj | Ne demek | Ne yapmalısın |
|-------|----------|---------------|
| Bu klasör henüz versiyon takibine açılmamış | `/init` çalıştırılmamış | `/init` |
| Adın ve e-posta henüz ayarlanmamış | Kimlik girilmemiş | `/setup` |
| GitHub'da senin bilmediğin değişiklikler var | Başkası veya başka bilgisayar güncellemiş | `/pull` |
| Kaydedecek bir şey yok | Zaten her şey kaydedilmiş | Normal, devam et |
| Kaydedilmemiş değişikliklerin var | Stage edilmemiş dosyalar var | `/stage` → `/commit` |
| GitHub'daki proje bulunamadı | URL yanlış veya erişim yok | `/github <yeni-url>` |
| Başka bir gitBook penceresi açık olabilir | Kilit dosyası var | Diğer pencereyi kapat |
| İki farklı sürüm aynı satırı değiştirmiş | Merge çakışması | Dosyayı düzelt → `/stage` → `/commit` |
| Bu iki proje aynı geçmişten gelmiyor | İlgisiz iki repo birleştirilmeye çalışıldı | `/status` ile duruma bak |
| Beklenmedik bir sorun çıktı | Bilinmeyen hata | `/status` ile durumu kontrol et |

---

*gitBook — Git bilmeyenler için versiyon kontrolü.*

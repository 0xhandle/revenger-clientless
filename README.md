# revenger-clientless

Revenger Online World login sunucusuna oyun olmadan giriş yapıp sunucu listesini çeken komut satırı aracı.

```
$ revenger-clientless -user kullanici -password sifre
ID    GRUP      SUNUCU                           ADRES          PORT
1004  REVENGER  RevengerOnline - 1(PK-MERCHANT)  82.xx.xx.xxx   19001
2004  BERAMUS   BERAMUS - 1(PK-MERCHANT)         82.xx.xx.xxx   19002
```

## Gereksinimler

Go 1.24 veya üstü ve bir Revenger Online hesabı. Liste sadece giriş yapan hesaba gönderiliyor.

## Derleme

```
make build
./bin/revenger-clientless -user kullanici -password sifre
```

ya da direkt:

```
go run ./cmd/revenger-clientless -user kullanici -password sifre
```

## Kullanım

| Parametre | Varsayılan | Açıklama |
|-----------|------------|----------|
| `-user` | | Kullanıcı adı (zorunlu) |
| `-password` | | Şifre (zorunlu) |
| `-pattern` | | Yeni cihazda istenen desen, örn. `1,2,3,6,9` |
| `-device` | kullanıcı adından | 32 karakterlik cihaz kimliği |
| `-host` | `82.38.10.101` | Login sunucusu adresi |
| `-port` | `19101` | Login sunucusu portu |
| `-json` | `false` | Listeyi json olarak yazdırır |
| `-timeout` | `10s` | Bağlantı zaman aşımı |

### Desen ve cihaz kimliği

Oyun her girişte bir cihaz kimliği gönderiyor. Sunucu tanımadığı bir cihaz görünce desen istiyor, desen girildikten sonra o cihazı kaydediyor ve bir daha sormuyor.

`-device` verilmezse kimlik kullanıcı adından üretiliyor, yani aynı hesapla her çalıştırmada aynı değer gidiyor. İlk çalıştırmada desen isterse `-pattern` ile bir kere vermen yeterli:

```
revenger-clientless -user kullanici -password sifre -pattern 1,2,3,6,9
```

Desen, 3x3 kilit ekranındaki düğümlerin virgülle ayrılmış sırası. En az 5 düğüm olmalı.

## Protokol

Paketler `AA55 | uzunluk:u16le | veri | 55AA` şeklinde gidip geliyor. Sayılar little endian, yazılar `uzunluk:u16le` ve ardından UTF-16LE karakterler.

İstemcinin gönderdiği paketler `0x79E0` zarfıyla şifreleniyor:

```
E0 79 | seed:u32le | uzunluk:u32le | AES-128-CBC(veri)
```

Anahtar 12 sıfır baytın arkasına seed eklenerek oluşturuluyor, IV tamamen sıfır, padding de sıfırla yapılıyor. Seed paketin içinde açık gittiği için ortada gizli bir anahtar yok. Sunucunun cevapları şifresiz geliyor.

Giriş sırası:

1. `01FE` ping, sunucu aynı opcode ile cevap veriyor.
2. Şifreli `0003 0100` versiyon isteği, sunucu versiyon ve indirme adresleriyle dönüyor.
3. Şifreli `0001` giriş paketi: kullanıcı adı, şifre, cihaz kimliği, model, cihaz adı, ekran kartı bilgisi, ekran kartı belleği, sistem belleği ve tekrar cihaz kimliği.
4. Sunucu `0001 | sonuç:i32 | token` dönüyor, sonuç 1 ise giriş başarılı.
5. Cihaz yeniyse `0009 | adım:i32` geliyor, istemci `0009 | adım | desen` ile cevap veriyor.
6. Ardından `0002` sunucu listesi geliyor.

Liste `0002 | adet:i32` ve arkasından her kayıt için `id:i32 | ad | adres | port:u16 | üst_id:i32` şeklinde. Üst id'si `-1` olan kayıtlar grup başlığı, diğerleri o gruba ait sunucular.

## Test

```
make test
```

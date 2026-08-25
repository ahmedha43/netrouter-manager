# NetRouter OS

**NetRouter OS** هو مشروع نظام راوتر أصلي موجّه إلى `x86_64` في الإصدار الأول. يتكون المستودع من صورة Linux قابلة للإقلاع مبنية بـBuildroot، وخدمة إدارة مقيدة مكتوبة بـGo، وعميل Windows مستقل باسم `NetRouterManager.exe`. لا توجد لوحة ويب مطلوبة لإدارة النظام في مسار المنتج الأصلي.

> هذا ليس بديلاً كاملاً لمنتج راوتر تجاري. ما تم بناؤه واختباره موضح أدناه بدقة، وما يزال يحتاج إلى مختبر شبكي معزول أو عتاد مرجعي مذكور صراحةً ضمن الحدود المتبقية.

## المكوّنات

| المكوّن | المسار | الحالة الحالية |
| --- | --- | --- |
| صورة نظام x86_64 | `platform/buildroot-external/` | تُنتج نواة `bzImage` وrootfs وISO هجين قابل للإقلاع. |
| خدمة الإدارة | `native/cmd/netrouterd` | Unix socket محلي مقيد، وmTLS اختياري فقط عند توفير جميع الشهادات. |
| أداة التشخيص | `native/cmd/netrouterctl` | تستدعي `status` و`interfaces` عبر Unix socket. |
| عميل Windows | `native/cmd/netrouter-manager` | برنامج Fyne مستقل قابل للتجميع إلى EXE ويعرض الاتصال والحالة والواجهات. |
| بروتوكول الإدارة | `native/internal/protocol/` | JSON بإصدار صريح وعقد مشترك بين الخدمة والعميل. |
| الواجهة React القديمة | `client/` | نموذج واجهة تاريخي فقط، وليس مسار إدارة النظام الفعلي. |

## ما تم التحقق منه محلياً

| التحقق | الدليل |
| --- | --- |
| اختبارات Go | `cd native && go test ./...` نجح، ويغطي المدخلات، Unix socket، الإيقاف، mTLS، وقوالب DHCP/DNS وnftables. |
| سلامة تطبيق الإعدادات | DNS/DHCP والجدار الناري يتبعان مسار **stage → validate → commit**؛ فشل `dnsmasq --test` أو `nft -c` لا يستبدل الملف النشط. |
| خدمة داخل صورة OS | اختبار QEMU على rootfs منفصل أكد وجود `/run/netrouterd.sock` واستجابة `netrouterctl status` بحالة نظام حقيقية. |
| إقلاع الوسيط | اختبار QEMU المباشر من `rootfs.iso9660` وصل إلى `Starting netrouterd: OK` ثم شاشة Buildroot. |
| برنامج Windows | بُني `artifacts/windows/NetRouterManager.exe` كملف `PE32+` x86-64. لم يُشغّل بعد على جهاز Windows فعلي. |

## بنية المستودع

```text
native/                       خدمة Go، البروتوكول، عميل Windows، والاختبارات
platform/buildroot-external/  defconfig ووصفة netrouterd وإعدادات kernel وoverlay
scripts/                      أوامر بناء العناصر الأصلية وصورة x86_64
docs/                         المعمارية وقرار التقنية وملاحظات البحث
client/                       نموذج React تاريخي، لا يمثل المنتج الأصلي
```

## البناء المحلي

يتطلب بناء المكونات الأصلية Go 1.22 وMinGW-w64 لبناء عميل Windows. ينشئ السكربت binaries لينكس وEXE بدون تعديل شبكة الجهاز المضيف.

```bash
./scripts/build-native.sh
file artifacts/windows/NetRouterManager.exe
```

صورة النظام تعتمد على Buildroot `2026.05` وbr2-external كي تبقى تخصيصات NetRouter خارج شجرة Buildroot نفسها. هذا هو المسار الموصى به لتخصيص مشروع Buildroot. [1]

```bash
./scripts/build-image-x86_64.sh
ls -lh artifacts/buildroot-x86_64/images/
```

المخرج القابل للإقلاع هو `artifacts/buildroot-x86_64/images/rootfs.iso9660`. الصورة **hybrid** وتصلح لاختبار QEMU أو للكتابة لاحقاً إلى وسيط USB مخصص للاختبار؛ لا تكتبها إلى جهاز إنتاجي قبل اجتياز اختبار العتاد والنسخ الاحتياطي.

```bash
qemu-system-x86_64 \
  -m 512M -nographic -monitor none -boot d \
  -cdrom artifacts/buildroot-x86_64/images/rootfs.iso9660
```

ولإثبات تشغيل الخدمة محلياً من دون تعديل صورة المنتج، شغّل الاختبار الذي ينسخ rootfs إلى ملف مؤقت، يضيف نص إثبات مؤقت، ثم يتحقق من socket واستجابة `status` ويطفئ QEMU تلقائياً:

```bash
./scripts/test-image-qemu.sh
```

## حدود الأمان والنطاق

الخدمة لا تفتح TCP افتراضياً. لتفعيل إدارة عن بعد يجب تقديم `--listen` و`--tls-cert` و`--tls-key` و`--tls-client-ca` معاً؛ عندها تفرض TLS 1.3 وشهادة عميل موثقة. إجراءات تعديل الرابط والعنوان والمسار تتحقق من المدخلات ثم تستدعي أوامر Linux محددة فقط. إعدادات DHCP/DNS والجدار الناري تُنشأ في ملف مرحلي وتُتحقق قبل الاستبدال الذري للملف الدائم، لكن تطبيقها الحي لا ينبغي اختباره إلا في VM أو شبكة خاصة معزولة.

لا يزال التالي خارج ادعاء الإصدار 0.1: اختبار EXE على Windows حقيقي، اختبار بطاقات شبكة وقرص على mini-PC، معاملات إعدادات دائمة مع rollback، اختبار تكاملي DHCP/NAT/firewall في namespace أو VM، PPPoE وVPN وبروتوكولات routing، والنسخ الاحتياطي والتحديث ودعم ARM.

## CI

مسار GitHub Actions الأصلي يشغّل اختبارات Go، ويبني daemon وctl وWindows EXE، ويتحقق من نوع EXE ثم يرفع artifacts. لا يبني صورة Buildroot كاملة في CI حالياً بسبب كلفة البناء والتخزين؛ يبقى بناء ISO واختبار QEMU جزءاً من التحقق المحلي المنضبط.

## الملكية والاستقلالية

يستخدم المشروع اسماً وبروتوكولاً وأصولاً مستقلة. لا يعيد استخدام أي شعار أو كود أو أيقونات أو أصول رسومية أو بروتوكول مملوك لمنتج راوتر آخر.

## المراجع

[1] [Buildroot Manual — project-specific customizations outside Buildroot](https://buildroot.org/downloads/manual/customize-outside-br.txt)

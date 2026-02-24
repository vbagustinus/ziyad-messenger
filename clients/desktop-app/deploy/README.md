# Client Deploy Notes

Packaging desktop client:

```bash
cd clients/desktop-app
flutter pub get
flutter build macos --release
# atau
flutter build windows --release
```

Hasil build ada di folder `build/` platform masing-masing.

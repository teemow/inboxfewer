# Logo Assets

This directory contains the inboxfewer logo in various sizes for different use cases.

## Available Sizes

| File | Size | Use Case |
|------|------|----------|
| `logo-original.png` | 1181x1181 | Original high-resolution source file |
| `logo-512.png` | 512x512 | OAuth interstitial pages, landing pages |
| `logo-256.png` | 256x256 | README, documentation (also at repo root) |
| `logo-128.png` | 128x128 | Headers, navigation bars |
| `logo-64.png` | 64x64 | Small icons, badges |
| `logo-32.png` | 32x32 | Favicons, tiny icons |

## Usage

### README / Documentation
The 256px version is used in the repository root as `inboxfewer.png`.

### OAuth Interstitial (Production)
For the OAuth success page shown to users after authentication, use the 512px version via GitHub raw URL:

```
https://raw.githubusercontent.com/teemow/inboxfewer/main/assets/logo/logo-512.png
```

### Helm Configuration
```yaml
interstitial:
  logoURL: "https://raw.githubusercontent.com/teemow/inboxfewer/main/assets/logo/logo-512.png"
  logoAlt: "inboxfewer logo"
```

### Environment Variables
```bash
MCP_INTERSTITIAL_LOGO_URL=https://raw.githubusercontent.com/teemow/inboxfewer/main/assets/logo/logo-512.png
MCP_INTERSTITIAL_LOGO_ALT="inboxfewer logo"
```

## Regenerating Sizes

If the original logo changes, regenerate all sizes with:

```bash
cd assets/logo
magick logo-original.png -resize 512x512 logo-512.png
magick logo-original.png -resize 256x256 logo-256.png
magick logo-original.png -resize 128x128 logo-128.png
magick logo-original.png -resize 64x64 logo-64.png
magick logo-original.png -resize 32x32 logo-32.png
cp logo-256.png ../../inboxfewer.png
```


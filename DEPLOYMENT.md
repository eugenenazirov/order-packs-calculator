# Deployment Guide - Fly.io

This guide walks you through deploying the Order Packs Calculator to Fly.io's free tier.

## Why Fly.io?

- ✅ **Truly free tier** - 3 VMs with 256MB RAM each (sufficient for this app)
- ✅ **No cold starts** - Your app stays responsive 24/7
- ✅ **Auto HTTPS** - Free SSL certificates
- ✅ **Global edge network** - Fast response times worldwide
- ✅ **Docker-native** - Uses your existing Dockerfile
- ✅ **Perfect for demos** - Professional infrastructure for test tasks

## Prerequisites

1. **Fly.io Account** - Sign up at https://fly.io/app/sign-up
2. **Credit Card** - Required for verification (won't charge within free tier limits)
3. **Fly CLI** - Install the command-line tool

## Step 1: Install Fly CLI

### macOS (via Homebrew)
```bash
brew install flyctl
```

### macOS/Linux (via install script)
```bash
curl -L https://fly.io/install.sh | sh
```

### Windows (PowerShell)
```powershell
pwsh -Command "iwr https://fly.io/install.ps1 -useb | iex"
```

Verify installation:
```bash
flyctl version
```

## Step 2: Authenticate with Fly.io

```bash
flyctl auth login
```

This will open your browser to complete authentication.

## Step 3: Prepare Your Application

The repository already includes the necessary configuration files:
- `fly.toml` - Fly.io configuration
- `Dockerfile` - Multi-stage Docker build
- `.dockerignore` - Optimized build context

### (Optional) Customize Configuration

Edit `fly.toml` if needed:

```toml
app = "order-packs-calculator"  # Change to unique name if taken
primary_region = "ams"           # Change region: ams (Amsterdam), iad (Virginia), etc.
```

**Available regions:**
- `ams` - Amsterdam, Netherlands
- `cdg` - Paris, France
- `fra` - Frankfurt, Germany
- `lhr` - London, UK
- `iad` - Ashburn, Virginia (US)
- `sjc` - San Jose, California (US)
- `syd` - Sydney, Australia
- `nrt` - Tokyo, Japan
- `gru` - São Paulo, Brazil

Choose the region closest to your reviewer's location for best performance.

## Step 4: Deploy to Fly.io

From your project root directory:

```bash
# Launch the application (first-time deployment)
flyctl launch --config fly.toml --no-deploy

# This will:
# - Validate your fly.toml configuration
# - Create the app in Fly.io
# - NOT deploy yet (we'll do it manually next)

# Deploy the application
flyctl deploy
```

The deployment process will:
1. Build your Docker image
2. Push it to Fly.io's registry
3. Deploy to your selected region
4. Run health checks
5. Provide your application URL

**Expected output:**
```
==> Building image
...
==> Pushing image to fly
...
==> Monitoring deployment
...
 1 desired, 1 placed, 1 healthy, 0 unhealthy
--> v0 deployed successfully
```

## Step 5: Access Your Application

After successful deployment:

```bash
# Open your app in the browser
flyctl open

# Or get the URL
flyctl info
```

Your application will be available at:
```
https://order-packs-calculator.fly.dev
```

(Or your custom app name if you changed it)

## Step 6: Verify Deployment

Test the deployed application:

```bash
# Check health endpoint
curl https://order-packs-calculator.fly.dev/api/health

# Test calculation endpoint
curl -X POST https://order-packs-calculator.fly.dev/api/calculate \
  -H "Content-Type: application/json" \
  -d '{"items": 251}'

# Expected response:
# {"items":251,"packs":{"250":1,"500":0,"1000":0,"2000":0,"5000":0}}
```

## Useful Commands

### View Logs
```bash
# Real-time logs
flyctl logs

# Follow logs (like tail -f)
flyctl logs -a order-packs-calculator
```

### Check Application Status
```bash
flyctl status
```

### View Application Info
```bash
flyctl info
```

### Access Monitoring Dashboard
```bash
flyctl dashboard
```

Or visit: https://fly.io/apps/order-packs-calculator

### Scale Resources (if needed)
```bash
# Scale to different VM size (still within free tier)
flyctl scale vm shared-cpu-1x --memory 512

# Scale number of instances
flyctl scale count 2
```

### Update Environment Variables
```bash
# Update pack sizes
flyctl secrets set PACK_SIZES="100,200,500,1000"

# Update rate limiting
flyctl secrets set RATE_LIMIT_RPS="50"
flyctl secrets set RATE_LIMIT_BURST="100"
```

### Redeploy After Changes
```bash
# After making code changes
flyctl deploy

# Force rebuild (skip cache)
flyctl deploy --no-cache
```

## Troubleshooting

### Deployment Failed

**Check logs:**
```bash
flyctl logs
```

**Common issues:**
1. **App name taken:** Change `app` name in `fly.toml`
2. **Region unavailable:** Change `primary_region` in `fly.toml`
3. **Build failure:** Ensure Docker builds locally first: `docker build -t test .`

### Health Check Failing

If deployment shows unhealthy status:

```bash
# Check detailed status
flyctl status --all

# SSH into the VM
flyctl ssh console

# Inside VM, check if app is running
ps aux | grep pack-calculator
wget -qO- http://127.0.0.1:8080/api/health
```

### Application Not Responding

```bash
# Restart the application
flyctl apps restart order-packs-calculator

# Force redeploy
flyctl deploy --strategy immediate
```

### View Resource Usage

```bash
# Check if within free tier limits
flyctl status --all
```

Free tier includes:
- Up to 3 shared-cpu-1x VMs (256MB RAM each)
- 3GB persistent volume storage
- 160GB outbound data transfer

## Cleanup (After Demo)

To avoid any charges after your demo period:

```bash
# Destroy the application
flyctl apps destroy order-packs-calculator

# Confirm by typing the app name when prompted
```

**Note:** This permanently deletes your application and all its data.

## Cost Monitoring

Your application should stay within the free tier, but to be safe:

1. **Monitor usage:** https://fly.io/dashboard/personal/billing
2. **Set up alerts:** Fly.io will email you if approaching paid usage
3. **Check monthly:** Review your usage in the dashboard

**Expected usage for this app:**
- 1 VM × 256MB = Well within free tier ✅
- Minimal data transfer for demos ✅
- No database or persistent storage ✅

## Additional Resources

- **Fly.io Documentation:** https://fly.io/docs/
- **Pricing:** https://fly.io/docs/about/pricing/
- **Support:** https://community.fly.io/

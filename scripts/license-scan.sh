#!/bin/bash

# License Scanner for terraform-provider-entitle
# This script scans all dependencies and checks for license compatibility with MPL-2.0

set -e

PROJECT_LICENSE="MPL-2.0"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
OUTPUT_FILE="${SCRIPT_DIR}/license-scan-results.csv"
REPORT_FILE="${SCRIPT_DIR}/license-compatibility-report.txt"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}🔍 Starting License Scan for terraform-provider-entitle${NC}"
echo "=================================================="

# Check if go-licenses is installed
if ! command -v go-licenses &> /dev/null; then
    echo -e "${YELLOW}⚠️  go-licenses not found. Installing...${NC}"
    go install github.com/google/go-licenses@latest
    export PATH=$PATH:~/go/bin
fi

# Run the license scan
echo -e "${BLUE}📊 Scanning dependencies...${NC}"
go-licenses csv . > "$OUTPUT_FILE" 2>/dev/null || {
    echo -e "${RED}❌ License scan failed${NC}"
    exit 1
}

# Analyze results
echo -e "${BLUE}📋 Analyzing license compatibility...${NC}"

# Compatible licenses with MPL-2.0
COMPATIBLE_LICENSES=("MPL-2.0" "MIT" "BSD-2-Clause" "BSD-3-Clause" "Apache-2.0" "ISC" "Unlicense")

# Potentially problematic licenses
PROBLEMATIC_LICENSES=("GPL-2.0" "GPL-3.0" "AGPL-3.0" "LGPL-2.1" "LGPL-3.0" "SSPL-1.0" "Commons-Clause")

# Start report
cat > "$REPORT_FILE" << EOF
License Compatibility Report for terraform-provider-entitle
Generated: $(date)
Project License: $PROJECT_LICENSE

SUMMARY:
EOF

# Count licenses
total_packages=$(wc -l < "$OUTPUT_FILE")
echo "Total packages scanned: $total_packages" >> "$REPORT_FILE"

declare -A license_counts
declare -A license_packages

# Parse CSV and count licenses
while IFS=',' read -r package url license; do
    # Skip warning lines that might start with package names
    if [[ $package =~ ^W[0-9] ]] || [[ $package =~ ^E[0-9] ]]; then
        continue
    fi
    
    license_counts["$license"]=$((${license_counts["$license"]} + 1))
    if [[ -z "${license_packages["$license"]}" ]]; then
        license_packages["$license"]="$package"
    else
        license_packages["$license"]="${license_packages["$license"]}|$package"
    fi
done < "$OUTPUT_FILE"

echo "" >> "$REPORT_FILE"
echo "LICENSE DISTRIBUTION:" >> "$REPORT_FILE"

compatible_count=0
problematic_count=0
unknown_count=0

for license in "${!license_counts[@]}"; do
    count=${license_counts["$license"]}
    echo "  $license: $count packages" >> "$REPORT_FILE"
    
    # Check compatibility
    is_compatible=false
    is_problematic=false
    
    for comp_license in "${COMPATIBLE_LICENSES[@]}"; do
        if [[ "$license" == "$comp_license" ]]; then
            is_compatible=true
            compatible_count=$((compatible_count + count))
            break
        fi
    done
    
    if [[ "$is_compatible" == false ]]; then
        for prob_license in "${PROBLEMATIC_LICENSES[@]}"; do
            if [[ "$license" == "$prob_license" ]]; then
                is_problematic=true
                problematic_count=$((problematic_count + count))
                break
            fi
        done
    fi
    
    if [[ "$is_compatible" == false && "$is_problematic" == false ]]; then
        unknown_count=$((unknown_count + count))
    fi
done

# Summary
echo "" >> "$REPORT_FILE"
echo "COMPATIBILITY ANALYSIS:" >> "$REPORT_FILE"
echo "  ✅ Compatible packages: $compatible_count" >> "$REPORT_FILE"
echo "  ⚠️  Potentially problematic: $problematic_count" >> "$REPORT_FILE"
echo "  ❓ Unknown/requires review: $unknown_count" >> "$REPORT_FILE"

# Detailed analysis
echo "" >> "$REPORT_FILE"
echo "DETAILED ANALYSIS:" >> "$REPORT_FILE"

echo "" >> "$REPORT_FILE"
echo "✅ COMPATIBLE LICENSES:" >> "$REPORT_FILE"
for license in "${!license_counts[@]}"; do
    for comp_license in "${COMPATIBLE_LICENSES[@]}"; do
        if [[ "$license" == "$comp_license" ]]; then
            count=${license_counts["$license"]}
            echo "  $license ($count packages)" >> "$REPORT_FILE"
            IFS='|' read -ra packages <<< "${license_packages["$license"]}"
            for package in "${packages[@]}"; do
                echo "    - $package" >> "$REPORT_FILE"
            done
            break
        fi
    done
done

if [[ $problematic_count -gt 0 ]]; then
    echo "" >> "$REPORT_FILE"
    echo "⚠️  POTENTIALLY PROBLEMATIC LICENSES:" >> "$REPORT_FILE"
    for license in "${!license_counts[@]}"; do
        for prob_license in "${PROBLEMATIC_LICENSES[@]}"; do
            if [[ "$license" == "$prob_license" ]]; then
                count=${license_counts["$license"]}
                echo "  $license ($count packages) - REQUIRES REVIEW" >> "$REPORT_FILE"
                IFS='|' read -ra packages <<< "${license_packages["$license"]}"
                for package in "${packages[@]}"; do
                    echo "    - $package" >> "$REPORT_FILE"
                done
                break
            fi
        done
    done
fi

if [[ $unknown_count -gt 0 ]]; then
    echo "" >> "$REPORT_FILE"
    echo "❓ UNKNOWN/UNRECOGNIZED LICENSES:" >> "$REPORT_FILE"
    for license in "${!license_counts[@]}"; do
        is_known=false
        for comp_license in "${COMPATIBLE_LICENSES[@]}" "${PROBLEMATIC_LICENSES[@]}"; do
            if [[ "$license" == "$comp_license" ]]; then
                is_known=true
                break
            fi
        done
        
        if [[ "$is_known" == false ]]; then
            count=${license_counts["$license"]}
            echo "  $license ($count packages) - MANUAL REVIEW NEEDED" >> "$REPORT_FILE"
            IFS='|' read -ra packages <<< "${license_packages["$license"]}"
            for package in "${packages[@]}"; do
                echo "    - $package" >> "$REPORT_FILE"
            done
        fi
    done
fi

# Print results to console
echo -e "${GREEN}📄 Results Summary:${NC}"
echo "  Compatible packages: $compatible_count"
if [[ $problematic_count -gt 0 ]]; then
    echo -e "  ${RED}Potentially problematic: $problematic_count${NC}"
fi
if [[ $unknown_count -gt 0 ]]; then
    echo -e "  ${YELLOW}Unknown/requires review: $unknown_count${NC}"
fi

echo ""
echo -e "${BLUE}📁 Output files:${NC}"
echo "  📊 Raw scan results: $OUTPUT_FILE"
echo "  📋 Compatibility report: $REPORT_FILE"

# Exit codes
if [[ $problematic_count -gt 0 ]]; then
    echo -e "${RED}❌ FAILED: Potentially problematic licenses found${NC}"
    exit 1
elif [[ $unknown_count -gt 0 ]]; then
    echo -e "${YELLOW}⚠️  WARNING: Unknown licenses require manual review${NC}"
    exit 2
else
    echo -e "${GREEN}✅ SUCCESS: All licenses are compatible with $PROJECT_LICENSE${NC}"
    exit 0
fi
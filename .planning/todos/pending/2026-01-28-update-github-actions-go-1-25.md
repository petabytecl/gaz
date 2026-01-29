---
created: 2026-01-28T22:43
title: Update GitHub Actions to use Go 1.25 only
area: tooling
files:
  - .github/workflows/ci.yml
---

## Problem

The project currently references multiple Go versions (1.22, 1.23, 1.24) in GitHub Actions. We need to standardize on using only Go 1.25.

## Solution

Modify .github/workflows/ci.yml to remove older Go versions from the matrix/setup and ensure only Go 1.25 is used.

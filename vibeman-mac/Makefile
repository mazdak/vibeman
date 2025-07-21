# Vibeman macOS App Makefile

APP_NAME = Vibeman
BUNDLE_ID = com.vibeman.app
BUILD_DIR = .build
APP_DIR = $(BUILD_DIR)/$(APP_NAME).app
EXECUTABLE = $(BUILD_DIR)/debug/$(APP_NAME)

# Default target
.PHONY: all
all: build

# Build the Swift package
.PHONY: build
build:
	swift build

# Build for release
.PHONY: build-release
build-release:
	swift build -c release

# Quick development build
.PHONY: dev
dev:
	./build-dev.sh

# Run the app directly (development)
.PHONY: run
run: build
	swift run

# Create the macOS app bundle
.PHONY: app
app: build-release
	@echo "Creating app bundle..."
	@rm -rf $(APP_DIR)
	@mkdir -p $(APP_DIR)/Contents/MacOS
	@mkdir -p $(APP_DIR)/Contents/Resources
	
	# Copy executable
	@cp $(BUILD_DIR)/release/$(APP_NAME) $(APP_DIR)/Contents/MacOS/
	
	# Copy Info.plist
	@cp Sources/Info.plist $(APP_DIR)/Contents/
	
	# Copy Assets
	@cp -r Sources/Assets.xcassets $(APP_DIR)/Contents/Resources/
	
	# Copy vibeman executable (if available)
	@if [ -f ../vibeman ]; then \
		cp ../vibeman $(APP_DIR)/Contents/Resources/; \
	fi
	
	# Copy web app (if available)
	@if [ -d ../vibeman-web ]; then \
		cp -r ../vibeman-web $(APP_DIR)/Contents/Resources/; \
	fi
	
	@echo "App bundle created at $(APP_DIR)"

# Run the app directly (for development)
.PHONY: run
run: build
	$(EXECUTABLE)

# Run tests
.PHONY: test
test:
	swift test

# Clean build artifacts
.PHONY: clean
clean:
	swift package clean
	rm -rf $(BUILD_DIR)

# Create a distributable DMG (requires create-dmg)
.PHONY: dmg
dmg: app
	@echo "Creating DMG..."
	@if command -v create-dmg >/dev/null 2>&1; then \
		create-dmg \
			--volname "$(APP_NAME)" \
			--volicon "$(APP_DIR)/Contents/Resources/Assets.xcassets/AppIcon.appiconset" \
			--window-pos 200 120 \
			--window-size 600 300 \
			--icon-size 100 \
			--icon "$(APP_NAME).app" 175 120 \
			--hide-extension "$(APP_NAME).app" \
			--app-drop-link 425 120 \
			"$(BUILD_DIR)/$(APP_NAME).dmg" \
			"$(APP_DIR)"; \
	else \
		echo "create-dmg not found. Install with: brew install create-dmg"; \
	fi

# Install create-dmg if needed
.PHONY: install-dmg-deps
install-dmg-deps:
	@if ! command -v create-dmg >/dev/null 2>&1; then \
		echo "Installing create-dmg..."; \
		brew install create-dmg; \
	fi

# Show help
.PHONY: help
help:
	@echo "Available targets:"
	@echo "  build          - Build the Swift package"
	@echo "  build-release  - Build for release"
	@echo "  app           - Create macOS app bundle"
	@echo "  run           - Run the app directly"
	@echo "  test          - Run tests"
	@echo "  clean         - Clean build artifacts"
	@echo "  dmg           - Create distributable DMG"
	@echo "  help          - Show this help"
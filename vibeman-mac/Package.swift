// swift-tools-version: 5.9

import PackageDescription

let package = Package(
    name: "Vibeman",
    platforms: [
        .macOS(.v12)
    ],
    products: [
        .executable(
            name: "Vibeman",
            targets: ["Vibeman"]
        ),
        .library(
            name: "VibemanKit",
            targets: ["VibemanKit"]
        ),
    ],
    dependencies: [
        .package(url: "https://github.com/sparkle-project/Sparkle", from: "2.0.0"),
        .package(url: "https://github.com/LebJe/TOMLKit", from: "0.5.0"),
    ],
    targets: [
        .executableTarget(
            name: "Vibeman",
            dependencies: [
                "VibemanKit",
                .product(name: "Sparkle", package: "Sparkle"),
            ],
            path: "Sources",
            exclude: ["VibemanKit/"],
            sources: ["VibemanApp.swift", "ProcessManager.swift"],
            resources: [
                .copy("Vibeman-Menu-Icon-Template.png"),
                .process("Assets.xcassets")
            ]
        ),
        .target(
            name: "VibemanKit",
            dependencies: [
                .product(name: "TOMLKit", package: "TOMLKit"),
            ],
            path: "Sources/VibemanKit"
        ),
        .testTarget(
            name: "VibemanTests",
            dependencies: ["VibemanKit"],
            path: "Tests"
        ),
    ]
)
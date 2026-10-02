// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0.
import Foundation
import Darwin

@main struct ExportTests {
    static func main() throws {
        let manager = FileManager.default
        let directory = manager.temporaryDirectory.appendingPathComponent("orangecount-export-\(UUID().uuidString)")
        try manager.createDirectory(at: directory, withIntermediateDirectories: true)
        defer { try? manager.removeItem(at: directory) }
        let source = directory.appendingPathComponent("main.bean")
        let target = directory.appendingPathComponent("result.csv")
        try Data("source".utf8).write(to: source)
        try Data("old".utf8).write(to: target)
        let capture = try AnalysisFileExport(content: Data("v\n7\n".utf8), protectedPaths: [source.path])
        try capture.save(to: target)
        guard try Data(contentsOf: target) == Data("v\n7\n".utf8) else { fatalError("replacement lost captured content") }
        let symlink = directory.appendingPathComponent("symlink.csv")
        let hardlink = directory.appendingPathComponent("hardlink.csv")
        try manager.createSymbolicLink(at: symlink, withDestinationURL: source)
        try manager.linkItem(at: source, to: hardlink)
        for forbidden in [source, symlink, hardlink] {
            do { try capture.save(to: forbidden); fatalError("protected target accepted") }
            catch is AnalysisFileExportError { }
        }
        guard try Data(contentsOf: source) == Data("source".utf8) else { fatalError("source changed") }
        let moved = directory.appendingPathComponent("moved.bean")
        try manager.moveItem(at: source, to: moved)
        do { try capture.save(to: hardlink); fatalError("retained protected inode lost after source move") }
        catch AnalysisFileExportError.protectedTarget { }
        guard try Data(contentsOf: moved) == Data("source".utf8) else { fatalError("moved source changed") }
        let declaredMissing = directory.appendingPathComponent("missing.pdf")
        let missingCapture = try AnalysisFileExport(content: Data("export".utf8), protectedPaths: [declaredMissing.path])
        do { try missingCapture.save(to: declaredMissing); fatalError("missing attachment overwritten") }
        catch AnalysisFileExportError.protectedTarget { }
        let missing = directory.appendingPathComponent("missing/out.csv")
        do { try capture.save(to: missing); fatalError("missing directory accepted") } catch { }
        // Force the real POSIX writer to fail after a partial temporary write.
        var originalLimit = rlimit()
        guard getrlimit(RLIMIT_FSIZE, &originalLimit) == 0 else { fatalError("getrlimit failed") }
        var limit = originalLimit; limit.rlim_cur = 1
        let originalSignal = signal(SIGXFSZ, SIG_IGN)
        guard setrlimit(RLIMIT_FSIZE, &limit) == 0 else { fatalError("setrlimit failed") }
        do {
            defer { setrlimit(RLIMIT_FSIZE, &originalLimit); signal(SIGXFSZ, originalSignal) }
            do { try capture.save(to: target); fatalError("partial write failure accepted") }
            catch AnalysisFileExportError.filesystem(let code) {
                guard code == EFBIG else { fatalError("unexpected write error") }
            }
        }
        guard try Data(contentsOf: target) == Data("v\n7\n".utf8) else { fatalError("failed temporary write damaged existing target") }
        guard try manager.contentsOfDirectory(atPath: directory.path).allSatisfy({ !$0.hasPrefix(".orangecount-export-") }) else { fatalError("temporary file leaked") }
        print("PASS: captured file replacement, source/symlink/hardlink/moved-inode/missing-attachment rejection, invalid directory and temporary cleanup")
        print("PASS: actual partial POSIX write failure preserves existing target and removes temporary file")
    }
}

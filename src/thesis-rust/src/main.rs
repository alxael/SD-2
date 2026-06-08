use std::env;
use std::fs::{self, File};
use std::io::{self, BufWriter, Write};
use std::path::{Path, PathBuf};
use std::time::Instant;

use rand::RngCore;
use rayon::ThreadPoolBuilder;

mod hash;
use hash::hash;

const OUTPUT_SIZE: usize = 16; // 128-bit digest

// configurable tree-construction parameters (must match thesis-go)
const LEAF_SIZE: usize = 256 * hash::BITRATE; // 16 KB per leaf
const TREE_FANOUT: usize = 8; // children per internal node

const TEST_COUNT: usize = 50;
const INPUT_SIZES: [usize; 6] = [
    1 << 18,
    1 << 20,
    1 << 22,
    1 << 24,
    1 << 26,
    1 << 28,
];
const CORE_COUNTS: [usize; 5] = [1, 2, 4, 8, 16];

fn run_speed_test(input_bytes: usize) -> u128 {
    let mut input = vec![0u8; input_bytes];
    rand::thread_rng().fill_bytes(&mut input);

    let start = Instant::now();
    let _ = hash(&input, OUTPUT_SIZE, LEAF_SIZE, TREE_FANOUT);
    start.elapsed().as_nanos()
}

fn generate_speed_report(output_path: &Path) -> io::Result<()> {
    if let Some(parent) = output_path.parent() {
        if !parent.as_os_str().is_empty() {
            fs::create_dir_all(parent)?;
        }
    }

    let file = File::create(output_path)?;
    let mut writer = BufWriter::new(file);

    let mut header = String::from("inputMegabytes");
    for &cores in CORE_COUNTS.iter() {
        header.push_str(&format!(",maxMBPerSecond{}Core", cores));
    }
    writeln!(writer, "{}", header)?;

    for &input_bytes in INPUT_SIZES.iter() {
        let megabytes = (input_bytes as f64) / (1024.0 * 1024.0);
        let mut row = format!("{:.6}", megabytes);

        for &cores in CORE_COUNTS.iter() {
            let pool = ThreadPoolBuilder::new()
                .num_threads(cores)
                .build()
                .expect("could not build thread pool");

            let results: Vec<u128> = pool.install(|| {
                (0..TEST_COUNT).map(|_| run_speed_test(input_bytes)).collect()
            });

            let max_nanoseconds = *results.iter().max().unwrap_or(&0);
            let max_seconds = (max_nanoseconds as f64) / 1e9;
            let max_mb_per_second = if max_seconds > 0.0 {
                megabytes / max_seconds
            } else {
                0.0
            };

            row.push_str(&format!(",{:.4}", max_mb_per_second));
        }

        writeln!(writer, "{}", row)?;
    }

    writer.flush()
}

fn main() {
    let args: Vec<String> = env::args().collect();
    let output_path = if args.len() >= 2 {
        PathBuf::from(&args[1])
    } else {
        PathBuf::from("reports/test-speed.csv")
    };

    generate_speed_report(&output_path).expect("speed report failed");
    eprintln!("wrote {}", output_path.display());
}

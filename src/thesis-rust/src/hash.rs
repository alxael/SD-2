use rayon::prelude::*;

pub const STATE_SIZE: usize = 32;
pub const STATE_STEP: usize = 16;
pub const BITRATE: usize = STATE_STEP * 4; // 64 bytes

// tree construction parameters (must match thesis-go)
// leaf_size and tree_fanout are supplied per call to `hash`.
pub const NODE_EXTRA_ROUNDS: usize = 4;

pub const DOMAIN_LEAF: u32 = 0x4C454146; // "LEAF"
pub const DOMAIN_NODE: u32 = 0x4E4F4445; // "NODE"

#[inline(always)]
pub fn chaos_round(x: u32, y: u32) -> (u32, u32) {
    // baker's map
    let mask = ((x as i32) >> 31) as u32;
    let bx = (x ^ mask) << 1;
    let by = ((y ^ mask) >> 1) | (mask & (1u32 << 31));

    // gingerbreadman's map
    let abs_mask = ((bx as i32) >> 31) as u32;
    let abs_bx = (bx ^ abs_mask).wrapping_sub(abs_mask);
    let nx = abs_bx.wrapping_sub(by);
    let ny = bx;
    (nx, ny)
}

#[derive(Clone, Copy)]
pub struct State {
    pub data: [u32; STATE_SIZE],
}

impl State {
    #[inline]
    pub fn new() -> Self {
        State { data: [0u32; STATE_SIZE] }
    }

    #[inline(always)]
    fn diffuse(&mut self) {
        let d = &mut self.data;
        for half in 0..2 {
            let base = half * STATE_STEP;
            for index in 0..STATE_STEP {
                let next = (index + 1) % STATE_STEP;
                d[base + index] ^= d[base + next].rotate_left(13);
            }
        }
        for half in 0..2 {
            let base = half * STATE_STEP;
            for index in (0..STATE_STEP).rev() {
                let prev = (index + STATE_STEP - 1) % STATE_STEP;
                d[base + index] ^= d[base + prev].rotate_left(11);
            }
        }
        for index in 0..STATE_STEP {
            d[index] ^= d[STATE_STEP + index].rotate_left(7);
        }
        for index in 0..STATE_STEP {
            d[STATE_STEP + index] ^= d[index].rotate_left(19);
        }
    }

    #[inline(always)]
    fn chaos(&mut self) {
        for index in 0..STATE_STEP {
            let b_index = index + STATE_STEP;

            let mut x = self.data[index];
            let mut y = self.data[b_index];

            for _ in 0..20 {
                let (nx, ny) = chaos_round(x, y);
                x = nx;
                y = ny;
            }

            self.data[index] = x;
            self.data[b_index] = y;

            let target = (index + 1) % STATE_STEP + STATE_STEP;
            self.data[target] ^= x.rotate_left(9);
            self.data[target] ^= y.rotate_left(17);
        }
        self.diffuse();
    }

    #[inline(always)]
    fn absorb_block(&mut self, block: &[u8; BITRATE]) {
        for i in 0..STATE_STEP {
            let off = i * 4;
            let word = u32::from_be_bytes([
                block[off],
                block[off + 1],
                block[off + 2],
                block[off + 3],
            ]);
            self.data[i] ^= word;
        }
    }
}

#[inline]
fn pad(data: &[u8], size: usize) -> Vec<u8> {
    let mut pad_len = size - 1 - (data.len() % size);
    if pad_len < 1 {
        pad_len += size;
    }
    let total = data.len() + 1 + pad_len;
    let mut padded = Vec::with_capacity(total);
    padded.extend_from_slice(data);
    padded.push(0x80);
    padded.resize(total, 0);
    padded[total - 1] = 0x01;
    padded
}

#[inline]
fn new_domain_state(domain: u32, level: u32) -> State {
    let mut s = State::new();
    s.data[STATE_STEP] = domain;
    s.data[STATE_STEP + 1] = level;
    s
}

#[inline]
fn absorb_into(state: &mut State, data: &[u8]) {
    for block in data.chunks_exact(BITRATE) {
        let block_arr: &[u8; BITRATE] = block.try_into().unwrap();
        state.absorb_block(block_arr);
        state.chaos();
    }
}

fn hash_leaf(chunk: &[u8]) -> State {
    let mut state = new_domain_state(DOMAIN_LEAF, 0);
    absorb_into(&mut state, chunk);
    state
}

fn hash_node(children: &[State], level: u32) -> State {
    let mut state = new_domain_state(DOMAIN_NODE, level);
    let mut buf = vec![0u8; children.len() * STATE_STEP * 4];
    for (i, c) in children.iter().enumerate() {
        let off = i * STATE_STEP * 4;
        for j in 0..STATE_STEP {
            buf[off + j * 4..off + j * 4 + 4].copy_from_slice(&c.data[j].to_be_bytes());
        }
    }
    absorb_into(&mut state, &buf);
    for _ in 0..NODE_EXTRA_ROUNDS {
        state.chaos();
    }
    state
}

pub fn hash(input: &[u8], output_size: usize, leaf_size: usize, tree_fanout: usize) -> Vec<u8> {
    let padded = pad(input, BITRATE);

    let num_leaves = ((padded.len() + leaf_size - 1) / leaf_size).max(1);

    let mut nodes: Vec<State> = (0..num_leaves)
        .into_par_iter()
        .map(|i| {
            let start = i * leaf_size;
            let end = (start + leaf_size).min(padded.len());
            hash_leaf(&padded[start..end])
        })
        .collect();

    let mut level: u32 = 1;
    while nodes.len() > 1 {
        let group_count = (nodes.len() + tree_fanout - 1) / tree_fanout;
        let current = nodes;
        let next: Vec<State> = (0..group_count)
            .into_par_iter()
            .map(|i| {
                let start = i * tree_fanout;
                let end = (start + tree_fanout).min(current.len());
                hash_node(&current[start..end], level)
            })
            .collect();
        nodes = next;
        level += 1;
    }

    let mut final_state = nodes[0];

    let intermediate_rounds = 9 + (input.len() % 37 + output_size % 31) % 11;
    for _ in 0..intermediate_rounds {
        final_state.chaos();
    }

    let mut output: Vec<u8> = Vec::with_capacity(output_size + BITRATE);
    while output.len() < output_size {
        let mut buf = [0u8; STATE_STEP * 4];
        for i in 0..STATE_STEP {
            let off = i * 4;
            buf[off..off + 4].copy_from_slice(&final_state.data[i].to_be_bytes());
        }
        output.extend_from_slice(&buf);
        if output.len() < output_size {
            final_state.chaos();
        }
    }
    output.truncate(output_size);
    output
}

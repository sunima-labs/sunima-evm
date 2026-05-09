// FFI bridge between Go (Cosmos x/tfhe module) and tfhe-rs.
//
// Memory contract:
//   - All buffers returned to Go are allocated by Rust on the heap
//     and MUST be freed by Go calling `tfhe_free(ptr, len)`.
//   - Go-owned input buffers are read but never freed by Rust.
//   - Functions return 0 on success, non-zero error code on failure.
//
// This is the Stage 5.1 Week 1 hello-world: keygen, encrypt u64, add, decrypt.
// No bootstrap-bound ops (mul, compare) yet — Stage 5.4.

use std::os::raw::{c_int, c_uchar};
use std::slice;

use tfhe::prelude::*;
use tfhe::{generate_keys, set_server_key, ClientKey, ConfigBuilder, FheUint64, ServerKey};

// Error codes returned to Go.
const OK: c_int = 0;
const ERR_NULL_PTR: c_int = 1;
const ERR_DESERIALIZE: c_int = 2;
const ERR_SERIALIZE: c_int = 3;
const ERR_OPERATION: c_int = 4;

// ──────────────────────────────────────────────────────────────────────────
// Buffer ownership helpers
// ──────────────────────────────────────────────────────────────────────────

fn vec_to_buffer(v: Vec<u8>, out_ptr: *mut *mut c_uchar, out_len: *mut usize) {
    let mut boxed = v.into_boxed_slice();
    unsafe {
        *out_ptr = boxed.as_mut_ptr();
        *out_len = boxed.len();
    }
    std::mem::forget(boxed);
}

#[no_mangle]
pub extern "C" fn tfhe_free(ptr: *mut c_uchar, len: usize) {
    if ptr.is_null() {
        return;
    }
    unsafe {
        let _ = Vec::from_raw_parts(ptr, len, len);
    }
}

// ──────────────────────────────────────────────────────────────────────────
// Key generation
// ──────────────────────────────────────────────────────────────────────────

/// Generate a fresh (client_key, server_key) pair.
/// Both serialised via bincode and returned to Go.
#[no_mangle]
pub extern "C" fn tfhe_keygen(
    out_ck_ptr: *mut *mut c_uchar,
    out_ck_len: *mut usize,
    out_sk_ptr: *mut *mut c_uchar,
    out_sk_len: *mut usize,
) -> c_int {
    if out_ck_ptr.is_null() || out_sk_ptr.is_null() {
        return ERR_NULL_PTR;
    }
    let config = ConfigBuilder::default().build();
    let (ck, sk) = generate_keys(config);

    let ck_bytes = match bincode::serialize(&ck) {
        Ok(b) => b,
        Err(_) => return ERR_SERIALIZE,
    };
    let sk_bytes = match bincode::serialize(&sk) {
        Ok(b) => b,
        Err(_) => return ERR_SERIALIZE,
    };

    vec_to_buffer(ck_bytes, out_ck_ptr, out_ck_len);
    vec_to_buffer(sk_bytes, out_sk_ptr, out_sk_len);
    OK
}

// ──────────────────────────────────────────────────────────────────────────
// Encrypt / Decrypt
// ──────────────────────────────────────────────────────────────────────────

#[no_mangle]
pub extern "C" fn tfhe_encrypt_u64(
    ck_ptr: *const c_uchar,
    ck_len: usize,
    plaintext: u64,
    out_ct_ptr: *mut *mut c_uchar,
    out_ct_len: *mut usize,
) -> c_int {
    if ck_ptr.is_null() || out_ct_ptr.is_null() {
        return ERR_NULL_PTR;
    }
    let ck_slice = unsafe { slice::from_raw_parts(ck_ptr, ck_len) };
    let ck: ClientKey = match bincode::deserialize(ck_slice) {
        Ok(k) => k,
        Err(_) => return ERR_DESERIALIZE,
    };

    let ct = FheUint64::encrypt(plaintext, &ck);
    let bytes = match bincode::serialize(&ct) {
        Ok(b) => b,
        Err(_) => return ERR_SERIALIZE,
    };
    vec_to_buffer(bytes, out_ct_ptr, out_ct_len);
    OK
}

#[no_mangle]
pub extern "C" fn tfhe_decrypt_u64(
    ck_ptr: *const c_uchar,
    ck_len: usize,
    ct_ptr: *const c_uchar,
    ct_len: usize,
    out_plain: *mut u64,
) -> c_int {
    if ck_ptr.is_null() || ct_ptr.is_null() || out_plain.is_null() {
        return ERR_NULL_PTR;
    }
    let ck_slice = unsafe { slice::from_raw_parts(ck_ptr, ck_len) };
    let ct_slice = unsafe { slice::from_raw_parts(ct_ptr, ct_len) };

    let ck: ClientKey = match bincode::deserialize(ck_slice) {
        Ok(k) => k,
        Err(_) => return ERR_DESERIALIZE,
    };
    let ct: FheUint64 = match bincode::deserialize(ct_slice) {
        Ok(c) => c,
        Err(_) => return ERR_DESERIALIZE,
    };

    let plain: u64 = ct.decrypt(&ck);
    unsafe { *out_plain = plain };
    OK
}

// ──────────────────────────────────────────────────────────────────────────
// Homomorphic add (no bootstrap required for FheUint64 add)
// ──────────────────────────────────────────────────────────────────────────

#[no_mangle]
pub extern "C" fn tfhe_add_u64(
    sk_ptr: *const c_uchar,
    sk_len: usize,
    a_ptr: *const c_uchar,
    a_len: usize,
    b_ptr: *const c_uchar,
    b_len: usize,
    out_ct_ptr: *mut *mut c_uchar,
    out_ct_len: *mut usize,
) -> c_int {
    if sk_ptr.is_null() || a_ptr.is_null() || b_ptr.is_null() || out_ct_ptr.is_null() {
        return ERR_NULL_PTR;
    }
    let sk_slice = unsafe { slice::from_raw_parts(sk_ptr, sk_len) };
    let a_slice = unsafe { slice::from_raw_parts(a_ptr, a_len) };
    let b_slice = unsafe { slice::from_raw_parts(b_ptr, b_len) };

    let sk: ServerKey = match bincode::deserialize(sk_slice) {
        Ok(k) => k,
        Err(_) => return ERR_DESERIALIZE,
    };
    let a: FheUint64 = match bincode::deserialize(a_slice) {
        Ok(c) => c,
        Err(_) => return ERR_DESERIALIZE,
    };
    let b: FheUint64 = match bincode::deserialize(b_slice) {
        Ok(c) => c,
        Err(_) => return ERR_DESERIALIZE,
    };

    set_server_key(sk);
    let c = std::panic::catch_unwind(|| &a + &b);
    let c = match c {
        Ok(v) => v,
        Err(_) => return ERR_OPERATION,
    };

    let bytes = match bincode::serialize(&c) {
        Ok(b) => b,
        Err(_) => return ERR_SERIALIZE,
    };
    vec_to_buffer(bytes, out_ct_ptr, out_ct_len);
    OK
}

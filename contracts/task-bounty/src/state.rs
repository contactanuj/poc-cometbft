use cosmwasm_schema::cw_serde;
use cosmwasm_std::{Addr, Uint128};
use cw_storage_plus::{Item, Map};

#[cw_serde]
pub struct Config {
    pub admin: Addr,
    pub fee_collector: Addr,
    pub fee_pct: u64,
}

#[cw_serde]
pub struct Bounty {
    pub task_id: u64,
    pub funder: Addr,
    pub amount: Uint128,
    pub denom: String,
    pub multiplier_pct: u64,
    pub claimed: bool,
    pub claimer: Option<Addr>,
}

pub const CONFIG: Item<Config> = Item::new("config");
pub const BOUNTIES: Map<u64, Bounty> = Map::new("bounties");
pub const NEXT_BOUNTY_ID: Item<u64> = Item::new("next_bounty_id");

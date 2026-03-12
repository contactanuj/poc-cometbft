use cosmwasm_schema::{cw_serde, QueryResponses};
use crate::state::{Bounty, Config};

#[cw_serde]
pub struct InstantiateMsg {
    pub fee_collector: String,
    pub fee_pct: u64,
}

#[cw_serde]
pub enum ExecuteMsg {
    FundBounty { task_id: u64 },
    ClaimBounty { task_id: u64 },
    UpdateFee { fee_pct: u64 },
    AddBonus { task_id: u64, multiplier_pct: u64 },
}

#[cw_serde]
#[derive(QueryResponses)]
pub enum QueryMsg {
    #[returns(Config)]
    Config {},
    #[returns(Bounty)]
    Bounty { task_id: u64 },
    #[returns(ListBountiesResponse)]
    ListBounties {
        start_after: Option<u64>,
        limit: Option<u32>,
    },
}

#[cw_serde]
pub struct ListBountiesResponse {
    pub bounties: Vec<Bounty>,
}

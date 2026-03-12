use cosmwasm_std::StdError;
use thiserror::Error;

#[derive(Error, Debug)]
pub enum ContractError {
    #[error("{0}")]
    Std(#[from] StdError),

    #[error("unauthorized")]
    Unauthorized {},

    #[error("bounty already exists for task {task_id}")]
    BountyAlreadyExists { task_id: u64 },

    #[error("bounty not found for task {task_id}")]
    BountyNotFound { task_id: u64 },

    #[error("bounty already claimed for task {task_id}")]
    BountyAlreadyClaimed { task_id: u64 },

    #[error("no funds sent")]
    NoFunds {},

    #[error("must send exactly one denomination")]
    MultipleDenoms {},

    #[error("fee percentage must be <= 100")]
    InvalidFee {},
}

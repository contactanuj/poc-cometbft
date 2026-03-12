#[cfg(not(feature = "library"))]
use cosmwasm_std::entry_point;
use cosmwasm_std::{
    to_json_binary, BankMsg, Binary, Coin, Deps, DepsMut, Env, MessageInfo,
    Order, Response, StdResult, Uint128,
};
use cw2::set_contract_version;

use crate::error::ContractError;
use crate::msg::{ExecuteMsg, InstantiateMsg, ListBountiesResponse, QueryMsg};
use crate::state::{Bounty, Config, BOUNTIES, CONFIG};

const CONTRACT_NAME: &str = "crates.io:task-bounty";
const CONTRACT_VERSION: &str = env!("CARGO_PKG_VERSION");
const DEFAULT_LIMIT: u32 = 10;
const MAX_LIMIT: u32 = 30;

#[cfg_attr(not(feature = "library"), entry_point)]
pub fn instantiate(
    deps: DepsMut,
    _env: Env,
    info: MessageInfo,
    msg: InstantiateMsg,
) -> Result<Response, ContractError> {
    if msg.fee_pct > 100 {
        return Err(ContractError::InvalidFee {});
    }
    set_contract_version(deps.storage, CONTRACT_NAME, CONTRACT_VERSION)?;
    let config = Config {
        admin: info.sender.clone(),
        fee_collector: deps.api.addr_validate(&msg.fee_collector)?,
        fee_pct: msg.fee_pct,
    };
    CONFIG.save(deps.storage, &config)?;
    Ok(Response::new()
        .add_attribute("action", "instantiate")
        .add_attribute("admin", info.sender)
        .add_attribute("fee_collector", &msg.fee_collector)
        .add_attribute("fee_pct", msg.fee_pct.to_string())
        .add_attribute("contract", CONTRACT_NAME)
        .add_attribute("version", CONTRACT_VERSION))
}

#[cfg_attr(not(feature = "library"), entry_point)]
pub fn execute(
    deps: DepsMut,
    _env: Env,
    info: MessageInfo,
    msg: ExecuteMsg,
) -> Result<Response, ContractError> {
    match msg {
        ExecuteMsg::FundBounty { task_id } => execute_fund_bounty(deps, info, task_id),
        ExecuteMsg::ClaimBounty { task_id } => execute_claim_bounty(deps, info, task_id),
        ExecuteMsg::UpdateFee { fee_pct } => execute_update_fee(deps, info, fee_pct),
        ExecuteMsg::AddBonus {
            task_id,
            multiplier_pct,
        } => execute_add_bonus(deps, info, task_id, multiplier_pct),
    }
}

fn execute_fund_bounty(
    deps: DepsMut,
    info: MessageInfo,
    task_id: u64,
) -> Result<Response, ContractError> {
    if info.funds.is_empty() {
        return Err(ContractError::NoFunds {});
    }
    if info.funds.len() > 1 {
        return Err(ContractError::MultipleDenoms {});
    }
    if BOUNTIES.has(deps.storage, task_id) {
        return Err(ContractError::BountyAlreadyExists { task_id });
    }
    let coin = &info.funds[0];
    let bounty = Bounty {
        task_id,
        funder: info.sender.clone(),
        amount: coin.amount,
        denom: coin.denom.clone(),
        multiplier_pct: 100,
        claimed: false,
        claimer: None,
    };
    BOUNTIES.save(deps.storage, task_id, &bounty)?;
    Ok(Response::new()
        .add_attribute("action", "fund_bounty")
        .add_attribute("task_id", task_id.to_string())
        .add_attribute("funder", info.sender)
        .add_attribute("amount", coin.amount)
        .add_attribute("denom", &coin.denom)
        .add_attribute("multiplier_pct", "100"))
}

fn execute_claim_bounty(
    deps: DepsMut,
    info: MessageInfo,
    task_id: u64,
) -> Result<Response, ContractError> {
    let mut bounty = BOUNTIES
        .may_load(deps.storage, task_id)?
        .ok_or(ContractError::BountyNotFound { task_id })?;
    if bounty.claimed {
        return Err(ContractError::BountyAlreadyClaimed { task_id });
    }
    let config = CONFIG.load(deps.storage)?;
    let gross = bounty
        .amount
        .checked_mul(Uint128::from(bounty.multiplier_pct))?
        .checked_div(Uint128::from(100u64))?;
    let fee = gross
        .checked_mul(Uint128::from(config.fee_pct))?
        .checked_div(Uint128::from(100u64))?;
    let net = gross.checked_sub(fee)?;

    bounty.claimed = true;
    bounty.claimer = Some(info.sender.clone());
    BOUNTIES.save(deps.storage, task_id, &bounty)?;

    let mut msgs = vec![BankMsg::Send {
        to_address: info.sender.to_string(),
        amount: vec![Coin {
            denom: bounty.denom.clone(),
            amount: net,
        }],
    }];
    if !fee.is_zero() {
        msgs.push(BankMsg::Send {
            to_address: config.fee_collector.to_string(),
            amount: vec![Coin {
                denom: bounty.denom,
                amount: fee,
            }],
        });
    }

    Ok(Response::new()
        .add_messages(msgs)
        .add_attribute("action", "claim_bounty")
        .add_attribute("task_id", task_id.to_string())
        .add_attribute("claimer", info.sender)
        .add_attribute("gross", gross)
        .add_attribute("fee", fee)
        .add_attribute("net", net))
}

fn execute_update_fee(
    deps: DepsMut,
    info: MessageInfo,
    fee_pct: u64,
) -> Result<Response, ContractError> {
    let mut config = CONFIG.load(deps.storage)?;
    if info.sender != config.admin {
        return Err(ContractError::Unauthorized {});
    }
    if fee_pct > 100 {
        return Err(ContractError::InvalidFee {});
    }
    config.fee_pct = fee_pct;
    CONFIG.save(deps.storage, &config)?;
    Ok(Response::new()
        .add_attribute("action", "update_fee")
        .add_attribute("fee_pct", fee_pct.to_string()))
}

fn execute_add_bonus(
    deps: DepsMut,
    info: MessageInfo,
    task_id: u64,
    multiplier_pct: u64,
) -> Result<Response, ContractError> {
    let config = CONFIG.load(deps.storage)?;
    if info.sender != config.admin {
        return Err(ContractError::Unauthorized {});
    }
    let mut bounty = BOUNTIES
        .may_load(deps.storage, task_id)?
        .ok_or(ContractError::BountyNotFound { task_id })?;
    bounty.multiplier_pct = multiplier_pct;
    BOUNTIES.save(deps.storage, task_id, &bounty)?;
    Ok(Response::new()
        .add_attribute("action", "add_bonus")
        .add_attribute("task_id", task_id.to_string())
        .add_attribute("multiplier_pct", multiplier_pct.to_string()))
}

#[cfg_attr(not(feature = "library"), entry_point)]
pub fn query(deps: Deps, _env: Env, msg: QueryMsg) -> StdResult<Binary> {
    match msg {
        QueryMsg::Config {} => to_json_binary(&CONFIG.load(deps.storage)?),
        QueryMsg::Bounty { task_id } => to_json_binary(&BOUNTIES.load(deps.storage, task_id)?),
        QueryMsg::ListBounties { start_after, limit } => {
            let limit = limit.unwrap_or(DEFAULT_LIMIT).min(MAX_LIMIT) as usize;
            let start = start_after.map(cw_storage_plus::Bound::exclusive);
            let bounties: StdResult<Vec<_>> = BOUNTIES
                .range(deps.storage, start, None, Order::Ascending)
                .take(limit)
                .map(|item| item.map(|(_, b)| b))
                .collect();
            to_json_binary(&ListBountiesResponse {
                bounties: bounties?,
            })
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use cosmwasm_std::testing::{message_info, mock_dependencies, mock_env};
    use cosmwasm_std::{coins, Addr};

    fn setup_contract(deps: DepsMut) {
        let msg = InstantiateMsg {
            fee_collector: "fee_collector".to_string(),
            fee_pct: 10,
        };
        let info = message_info(&Addr::unchecked("admin"), &[]);
        instantiate(deps, mock_env(), info, msg).unwrap();
    }

    #[test]
    fn test_instantiate() {
        let mut deps = mock_dependencies();
        setup_contract(deps.as_mut());
        let config = CONFIG.load(deps.as_ref().storage).unwrap();
        assert_eq!(config.admin, Addr::unchecked("admin"));
        assert_eq!(config.fee_pct, 10);
    }

    #[test]
    fn test_fund_bounty() {
        let mut deps = mock_dependencies();
        setup_contract(deps.as_mut());
        let info = message_info(&Addr::unchecked("funder"), &coins(1000, "utoken"));
        let msg = ExecuteMsg::FundBounty { task_id: 1 };
        let res = execute(deps.as_mut(), mock_env(), info, msg).unwrap();
        assert_eq!(res.attributes.len(), 4);
        let bounty = BOUNTIES.load(deps.as_ref().storage, 1).unwrap();
        assert_eq!(bounty.amount, Uint128::new(1000));
        assert_eq!(bounty.denom, "utoken");
        assert!(!bounty.claimed);
    }

    #[test]
    fn test_fund_duplicate() {
        let mut deps = mock_dependencies();
        setup_contract(deps.as_mut());
        let info = message_info(&Addr::unchecked("funder"), &coins(1000, "utoken"));
        execute(
            deps.as_mut(),
            mock_env(),
            info.clone(),
            ExecuteMsg::FundBounty { task_id: 1 },
        )
        .unwrap();
        let err = execute(
            deps.as_mut(),
            mock_env(),
            info,
            ExecuteMsg::FundBounty { task_id: 1 },
        )
        .unwrap_err();
        assert!(matches!(err, ContractError::BountyAlreadyExists { .. }));
    }

    #[test]
    fn test_no_funds() {
        let mut deps = mock_dependencies();
        setup_contract(deps.as_mut());
        let info = message_info(&Addr::unchecked("funder"), &[]);
        let err = execute(
            deps.as_mut(),
            mock_env(),
            info,
            ExecuteMsg::FundBounty { task_id: 1 },
        )
        .unwrap_err();
        assert!(matches!(err, ContractError::NoFunds {}));
    }

    #[test]
    fn test_claim() {
        let mut deps = mock_dependencies();
        setup_contract(deps.as_mut());
        let fund_info = message_info(&Addr::unchecked("funder"), &coins(1000, "utoken"));
        execute(
            deps.as_mut(),
            mock_env(),
            fund_info,
            ExecuteMsg::FundBounty { task_id: 1 },
        )
        .unwrap();
        let claim_info = message_info(&Addr::unchecked("claimer"), &[]);
        let res = execute(
            deps.as_mut(),
            mock_env(),
            claim_info,
            ExecuteMsg::ClaimBounty { task_id: 1 },
        )
        .unwrap();
        // gross=1000, fee=100, net=900
        assert_eq!(res.messages.len(), 2);
        let bounty = BOUNTIES.load(deps.as_ref().storage, 1).unwrap();
        assert!(bounty.claimed);
        assert_eq!(bounty.claimer, Some(Addr::unchecked("claimer")));
    }

    #[test]
    fn test_claim_already_claimed() {
        let mut deps = mock_dependencies();
        setup_contract(deps.as_mut());
        let fund_info = message_info(&Addr::unchecked("funder"), &coins(1000, "utoken"));
        execute(
            deps.as_mut(),
            mock_env(),
            fund_info,
            ExecuteMsg::FundBounty { task_id: 1 },
        )
        .unwrap();
        let claim_info = message_info(&Addr::unchecked("claimer"), &[]);
        execute(
            deps.as_mut(),
            mock_env(),
            claim_info.clone(),
            ExecuteMsg::ClaimBounty { task_id: 1 },
        )
        .unwrap();
        let err = execute(
            deps.as_mut(),
            mock_env(),
            claim_info,
            ExecuteMsg::ClaimBounty { task_id: 1 },
        )
        .unwrap_err();
        assert!(matches!(err, ContractError::BountyAlreadyClaimed { .. }));
    }

    #[test]
    fn test_update_fee_unauthorized() {
        let mut deps = mock_dependencies();
        setup_contract(deps.as_mut());
        let info = message_info(&Addr::unchecked("random"), &[]);
        let err = execute(
            deps.as_mut(),
            mock_env(),
            info,
            ExecuteMsg::UpdateFee { fee_pct: 20 },
        )
        .unwrap_err();
        assert!(matches!(err, ContractError::Unauthorized {}));
    }

    #[test]
    fn test_add_bonus() {
        let mut deps = mock_dependencies();
        setup_contract(deps.as_mut());
        let fund_info = message_info(&Addr::unchecked("funder"), &coins(1000, "utoken"));
        execute(
            deps.as_mut(),
            mock_env(),
            fund_info,
            ExecuteMsg::FundBounty { task_id: 1 },
        )
        .unwrap();
        let admin_info = message_info(&Addr::unchecked("admin"), &[]);
        execute(
            deps.as_mut(),
            mock_env(),
            admin_info,
            ExecuteMsg::AddBonus {
                task_id: 1,
                multiplier_pct: 200,
            },
        )
        .unwrap();
        let bounty = BOUNTIES.load(deps.as_ref().storage, 1).unwrap();
        assert_eq!(bounty.multiplier_pct, 200);
    }

    #[test]
    fn test_list_pagination() {
        let mut deps = mock_dependencies();
        setup_contract(deps.as_mut());
        for i in 1..=5 {
            let info = message_info(&Addr::unchecked("funder"), &coins(100 * i, "utoken"));
            execute(
                deps.as_mut(),
                mock_env(),
                info,
                ExecuteMsg::FundBounty { task_id: i as u64 },
            )
            .unwrap();
        }
        let res = query(
            deps.as_ref(),
            mock_env(),
            QueryMsg::ListBounties {
                start_after: None,
                limit: Some(2),
            },
        )
        .unwrap();
        let list: ListBountiesResponse = cosmwasm_std::from_json(res).unwrap();
        assert_eq!(list.bounties.len(), 2);
        assert_eq!(list.bounties[0].task_id, 1);
        assert_eq!(list.bounties[1].task_id, 2);

        let res = query(
            deps.as_ref(),
            mock_env(),
            QueryMsg::ListBounties {
                start_after: Some(2),
                limit: Some(2),
            },
        )
        .unwrap();
        let list: ListBountiesResponse = cosmwasm_std::from_json(res).unwrap();
        assert_eq!(list.bounties.len(), 2);
        assert_eq!(list.bounties[0].task_id, 3);
    }
}

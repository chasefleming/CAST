import React, { useEffect, useRef, useState } from 'react';
import Blockies from 'react-blockies';
import { useVotingResults, useWindowDimensions } from 'hooks';
import useMediaQuery, { mediaMatchers } from 'hooks/useMediaQuery';
import { parseDateFromServer } from 'utils';
import { truncateAddress as truncate } from 'utils';
import { LinkOut } from './Svg';
import Tooltip from './Tooltip';

const BlockieWithAddress = React.forwardRef(
  ({ creatorAddr, isCoreCreator }, ref) => {
    const [addr, setAdd] = useState(creatorAddr);

    const { width } = useWindowDimensions();

    useEffect(() => {
      if (ref?.current.clientWidth <= 223 && creatorAddr === addr) {
        setAdd(truncate(creatorAddr));
      } else if (ref?.current.clientWidth > 223 && creatorAddr !== addr) {
        setAdd(creatorAddr);
      }
    }, [ref, width, creatorAddr, addr]);

    return (
      <div className="columns is-mobile m-0">
        <div className="column is-narrow is-flex is-align-items-center p-0">
          <Blockies
            seed={creatorAddr}
            size={6}
            scale={4}
            className="blockies"
          />
        </div>
        <div className="column px-2 py-0 is-flex flex-1 is-align-items-center">
          {addr}
        </div>
        {isCoreCreator && (
          <div
            className="column p-0 is-flex is-align-items-center is-justify-content-center-tablet subtitle is-size-7"
            style={{ fontFamily: 'Roboto Mono' }}
          >
            Core
          </div>
        )}
      </div>
    );
  }
);

const InfoBlock = ({ title, content, component }) => {
  const containerRef = useRef();
  // warn component consumer
  if (content && component) {
    console.warn('InfoBlock: please provide content or component');
  }
  return (
    <div
      className="columns is-mobile p-0 m-0 mb-5 small-text"
      ref={containerRef}
    >
      <div className="column p-0 is-flex is-align-items-center flex-1 has-text-grey is-4 is-5-desktop">
        {title}
      </div>
      <div
        className="column p-0 is-flex flex-1 is-align-items-center"
        style={{
          height: '1.5rem',
        }}
      >
        {content}
        {component && React.cloneElement(component, { ref: containerRef })}
      </div>
    </div>
  );
};

const Results = ({ voteResults }) => {
  const showViewMore = false;
  const options = Object.keys(voteResults);

  const totalVotes = options.reduce(
    (previousValue, currentValue) => previousValue + voteResults[currentValue],
    0
  );

  return (
    <>
      {options.map((option, index) => {
        const percentage =
          totalVotes === 0 || voteResults[option] === 0
            ? 0
            : ((100 * voteResults[option]) / totalVotes).toFixed(2);

        const optionText =
          option.length > 120 ? `${option.substring(0, 120)}...` : option;
        return (
          <div key={`result-item-${index}`} style={{ marginBottom: '2.5rem' }}>
            <div className="columns is-mobile mb-2">
              <div className="column small-text has-text-grey has-text-left word-break">
                {optionText}
              </div>
              <div className="column is-3 is-flex is-justify-content-flex-end small-text has-text-grey">
                {`${percentage}%`}
              </div>
            </div>
            <div
              style={{ height: 8, background: '#DCDCDC' }}
              className="has-background-grey-light rounded-lg"
            >
              <div
                className="rounded-lg"
                style={{
                  width: `${percentage}%`,
                  height: '100%',
                  background: '#747474',
                }}
              />
            </div>
          </div>
        );
      })}
      {showViewMore && (
        <div className="is-flex is-justify-content-start is-align-items-center">
          <button className="button is-white has-background-white-ter p-0">
            View more
          </button>
        </div>
      )}
    </>
  );
};

const WrapperSpacingTop = ({
  isMobileOnly,
  isTabletOnly,
  isDesktopOnly,
  children,
}) => {
  let classNames = '';
  if (isMobileOnly) {
    classNames = 'px-1 pb-0 pt-1';
  }
  if (isTabletOnly) {
    classNames = 'px-5 pb-0 pt-5';
  }
  if (isDesktopOnly) {
    classNames = 'px-6 pb-0 pt-6';
  }
  return <div className={classNames}>{children}</div>;
};

const WrapperSpacingBottom = ({
  isMobileOnly,
  isTabletOnly,
  isDesktopOnly,
  children,
}) => {
  let classNames = '';
  if (isMobileOnly) {
    classNames = 'px-1 pt-1 pb-1';
  }
  if (isTabletOnly) {
    classNames = 'px-5 pt-3 pb-4';
  }
  if (isDesktopOnly) {
    classNames = 'px-6 pt-2 pb-6';
  }
  return <div className={classNames}>{children}</div>;
};

const ProposalInformation = ({
  creatorAddr = '',
  strategyName = '',
  isCoreCreator = false,
  ipfs = '',
  ipfsUrl = '',
  startTime = '',
  endTime = '',
  proposalId = '',
  openStrategyModal = () => {},
  className = '',
}) => {
  const dateFormatConf = {
    day: 'numeric',
    hour: 'numeric',
    minute: 'numeric',
    month: 'short',
    year: 'numeric',
    hour12: true,
  };
  // stores navbar height calculated after component is mounted
  const [navbarHeight, setNavbarHeight] = useState(0);

  const isNotMobile = useMediaQuery();
  const isTabletOnly = useMediaQuery(mediaMatchers.tabletOnly);

  useEffect(() => {
    setNavbarHeight(document.querySelector('header').offsetHeight);
  }, []);

  const { height: windowHeight, width: windowWidth } = useWindowDimensions();

  // stores when user scrolls
  const [scroll, setScroll] = useState(0);

  // stores style to apply when info bar needs to be fixed
  const [fixedStyle, setFixedStyle] = useState(null);
  // ref of the panel info component
  const ref = useRef(null);
  // ref of the parent component
  const parentRef = useRef(null);
  // used to store return point
  const topRef = useRef({ pointStatic: null });

  const { isLoading: loadingVotingResults, data: votingResults } =
    useVotingResults(proposalId);

  // this effect watches for user scroll to make info panel fixed to navbar
  useEffect(() => {
    if (ref?.current && parentRef?.current) {
      const { top, height: infoPanelHeightSize } =
        ref?.current.getBoundingClientRect() || {};

      const { width } = parentRef?.current.getBoundingClientRect() || {};

      const winScroll =
        document.body.scrollTop || document.documentElement.scrollTop;

      // if window size is bigger that navbar size + info panel then apply fixed
      if (windowHeight > navbarHeight + infoPanelHeightSize) {
        // user scrolled down and panel is next to navbar
        // adding 4px so the nav bar sticks more smoothly
        if (top < navbarHeight + 4 && !fixedStyle) {
          // save reference where the element needs to go back
          if (!topRef.current.returnTop) {
            topRef.current.pointStatic = winScroll;
          }
          setFixedStyle({
            className: ' is-panel-fixed',
            // use width of parent component
            style: {
              width,
              top: navbarHeight,
            },
          });
        }
      }

      if (
        fixedStyle &&
        topRef.current.pointStatic &&
        topRef.current.pointStatic > winScroll
      ) {
        topRef.current.pointStatic = null;
        setFixedStyle(null);
      }
    }
  }, [scroll, fixedStyle, windowHeight, navbarHeight]);

  // this effect watches for window width resizes to change info panel width
  useEffect(() => {
    if (parentRef?.current) {
      const { width } = parentRef?.current.getBoundingClientRect() || {};

      // window was resized then change width of the component if fixed style was applied
      if (fixedStyle && fixedStyle.windowWidth !== windowWidth) {
        setFixedStyle((state) => ({
          ...state,
          windowWidth,
          style: { ...(state?.style ?? {}), width },
        }));
      }
    }
  }, [fixedStyle, windowWidth, navbarHeight]);

  function handleScroll() {
    setScroll(document.body.scrollTop || document.documentElement.scrollTop);
  }

  // this effect watches for window scrolling
  useEffect(() => {
    document.addEventListener('scroll', handleScroll);
    return () => document.removeEventListener('scroll', handleScroll);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  return (
    <div ref={parentRef}>
      <div
        className={`${className} rounded-sm${fixedStyle?.className || ''}`}
        ref={ref}
        style={fixedStyle?.style || {}}
      >
        <WrapperSpacingTop
          isMobileOnly={!isNotMobile}
          isDesktopOnly={isNotMobile && !isTabletOnly}
          isTabletOnly={isTabletOnly}
        >
          <p className="mb-5">Information</p>
          <InfoBlock
            title="Strategy"
            content={
              <div className="is-flex flex-1" onClick={openStrategyModal}>
                <div className="pr-2 cursor-pointer">{strategyName}</div>
              </div>
            }
          />
          <InfoBlock
            title={'Author'}
            component={
              <BlockieWithAddress
                creatorAddr={creatorAddr}
                isCoreCreator={isCoreCreator}
              />
            }
          />
          {ipfs && (
            <InfoBlock
              title={'IPFS'}
              content={
                <a
                  href={ipfsUrl}
                  rel="noopener noreferrer"
                  target="_blank"
                  className="button is-text p-0 small-text"
                  style={{ height: '2rem !important' }}
                >
                  <Tooltip
                    classNames="is-flex is-flex-grow-1 is-align-items-center"
                    position="top"
                    text="Open Ipfs link"
                  >
                    <p className="mr-2">{`${ipfs.substring(0, 8)}`}</p>
                    <LinkOut width="12" height="12" />
                  </Tooltip>
                </a>
              }
            />
          )}
          <InfoBlock
            title={'Start date'}
            content={parseDateFromServer(startTime).date.toLocaleString(
              undefined,
              dateFormatConf
            )}
          />
          <InfoBlock
            title={'End date'}
            content={parseDateFromServer(endTime).date.toLocaleString(
              undefined,
              dateFormatConf
            )}
          />
        </WrapperSpacingTop>
        <hr />
        <WrapperSpacingBottom
          isMobileOnly={!isNotMobile}
          isDesktopOnly={isNotMobile && !isTabletOnly}
          isTabletOnly={isTabletOnly}
        >
          <p className="mb-5">Current Results</p>
          {!loadingVotingResults && (
            <Results voteResults={votingResults?.results || []} />
          )}
        </WrapperSpacingBottom>
      </div>
    </div>
  );
};

export default ProposalInformation;

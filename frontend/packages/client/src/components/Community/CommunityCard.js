import React from 'react';
import Blockies from 'react-blockies';
import { Link } from 'react-router-dom';
import { useMediaQuery } from 'hooks';
import JoinCommunityButton from './JoinCommunityButton';

/**
 * CommunityCard will group communities on a row bases,
 * will use elementsPerRow to determine how many communities to render per row
 */
const CommunityCard = ({ logo, name, body, id, slug, hideJoin }) => {
  const descriptionStyle = {
    lineHeight: '1.5em',
    maxHeight: '3rem',
  };

  const isNotMobile = useMediaQuery();
  const avatarSize = isNotMobile
    ? {
        logo: { width: 96, height: 96 },
        blockie: { size: 10, scale: 9.6 },
        columnStyle: { maxHeight: '120px' },
      }
    : {
        logo: { width: 48, height: 48 },
        blockie: { size: 10, scale: 4.8 },
        columnStyle: { maxHeight: '72px' },
      };

  return (
    <>
      <Link to={`/community/${id}?tab=proposals`} style={{ color: 'inherit' }}>
        <div className="is-flex is-flex-grow-1 rounded-lg border-light p-5 p-3-mobile is-flex-direction-column transition-all community-card">
          <div className="columns is-multiline is-flex-grow-1 is-mobile">
            <div
              className="column is-narrow pr-2-mobile"
              style={avatarSize.columnStyle}
            >
              {logo ? (
                <div
                  className="border-light rounded-full"
                  style={{
                    ...avatarSize.logo,
                    backgroundImage: `url(${logo})`,
                    backgroundSize: 'cover',
                  }}
                />
              ) : (
                <Blockies
                  seed={slug ?? `seed-${id}`}
                  size={avatarSize.blockie.size}
                  scale={avatarSize.blockie.scale}
                  className="rounded-full"
                />
              )}
            </div>
            <div
              className="column pl-2-mobile pb-0-mobile is-flex is-flex-direction-column"
              style={
                isNotMobile
                  ? {
                      justifyContent: 'center',
                    }
                  : {}
              }
            >
              <div className="is-size-5 is-size-6-mobile mb-2 is-4 is-6-mobile pt-0-mobile line-clamp-2">
                {name}
              </div>
              <p
                className="has-text-grey small-text line-clamp-2"
                style={descriptionStyle}
              >
                {body}
              </p>
            </div>
            {!hideJoin && (
              <div className="column is-12-mobile p-0-mobile is-narrow-tablet is-flex is-flex-direction-column is-justify-content-start">
                <JoinCommunityButton communityId={id} />
              </div>
            )}
          </div>
        </div>
      </Link>
    </>
  );
};

export default CommunityCard;
